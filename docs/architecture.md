# Unseal — Architecture

This document describes how Unseal is put together and why. The behavioural
specification of the product lives in the original design document,
[`design.md`](../design.md); this document covers the Go implementation.

---

## 1. Layer model

Unseal follows the same six-layer shape as the original C# project, with the
layering enforced by import direction.

```
        ┌──────────────────────────────────────────────┐
   L5   │  app              cmd/unseal                 │  composition root
        ├──────────────────────────────────────────────┤
   L4   │  ui               (GOUI desktop widgets)     │  presentation
        ├──────────────────────────────────────────────┤
   L3   │  core  audible  decrypt  export  pipeline    │  domain
        ├──────────────────────────────────────────────┤
   L2   │  db                                         │  persistence
        ├──────────────────────────────────────────────┤
   L1   │  config                audible/model          │  shared contracts
        ├──────────────────────────────────────────────┤
   L0   │  logging   util   hw   version               │  foundation
        └──────────────────────────────────────────────┘
```

**Rules**

1. A package may only import from its own layer or a lower one.
2. L3 and below must never import `ui`. The domain code has no idea the
   graphical interface exists.
3. L0 and L1 must not import `db`, `audible`, or anything above them.
4. Only `app` and `cmd/unseal` may see several layers at once.
5. L4 talks to L3 through interfaces, never through concrete types.

These rules are checked by review, not by a linter, but they are what keeps
the protocol code testable without a display and the GUI code replaceable.

---

## 2. Package responsibilities

| Package | Responsibility |
|---------|----------------|
| `version` | Build identity — link-time injection with a `debug.ReadBuildInfo` fallback. |
| `logging` | Leveled logging, daily rolling files, level-gated lazy messages, the log/temp housekeeping pass. |
| `util` | Pure helpers — filename pruning, long-path normalisation, time rounding and formatting, CRC32, HTML to text. |
| `hw` | Stable machine identifiers, per platform, behind one interface. |
| `config` | The two-tier settings model with atomic writes and immediate persistence on change. |
| `db` | Entities, the SQLite context, ordered migrations, the repository, and the state aggregation rules. |
| `audible` | The wire protocol: registration, token refresh, ADP signing, library listing, catalogue lookup, licence acquisition and payload download. |
| `audible/model` | The JSON shapes Audible returns, and nothing else. |
| `decrypt` | The `Decryptor` interface and the ffmpeg backend that removes the DRM. |
| `export` | The `.aax` interchange format and its sidecar JSON files. |
| `pipeline` | The state machine, stage orchestration, throttling, progress and reconciliation. |
| `core` | The domain facade the user interface talks to. |
| `ui` | The GOUI desktop interface: windows, tables, split views, dialogs. |
| `app` | The composition root: wiring, settings, startup sequence. |

---

## 3. Key design decisions

### 3.1 The unit of work is a conversion

The domain model has three central entities and the distinction matters:

- A **book** is what Audible sells. It carries the marketing metadata — title,
  summary, rating, cover — and a delivery type.
- A **component** is a physical part of a multi-part book. Components inherit
  most metadata from their parent book.
- A **conversion** is a download-decrypt-export unit of work. Every book has
  exactly one "whole book" conversion and every component has exactly one
  "part N" conversion.

This is what lets the download grid show a flat list mixing whole-book rows and
part rows, and lets one code path in `pipeline` treat both identically.

`Conversion.AccountId` plus `Conversion.Region` is the tenancy key: the same
title can exist for several accounts and regions.

### 3.2 State ordinals are meaningful

`EConversionState` has sixteen states whose numeric order carries meaning —
several code paths compare against `local_locked` with `<` and `>` to decide
whether a download or a decryption is still required. New states must be
inserted with care, and the icon map and tooltip resources must be updated
alongside.

The `download` state is transient: it marks a row as staged in the interface.
`ApplicableState` resolves it back to whatever the persisted state was, so the
grid always shows the truth.

### 3.3 Decryption sits behind an interface

The original used the `AAXClean` NuGet package, which does the DRM removal
in-process. Go has no equivalent, so Unseal defines a `Decryptor` interface and
ships an ffmpeg backend that uses the native `aax` demuxer:

```sh
ffmpeg -audible_key <hex> -audible_iv <hex> -i in.aaxc -c copy out.m4b
```

A pure-Go implementation can be added later behind the same interface without
touching the pipeline. The trade-off is that ffmpeg becomes a runtime
dependency, which the original did not have — hence the bootstrap downloader
planned in milestone M8.

### 3.4 The credential store is encrypted, and that is not strong

The profile store holds bearer tokens, refresh tokens, the device private key
and the ADP token. It is encrypted with AES-256 using a key derived from
PBKDF2-HMAC-SHA256 over a token built from `ROT13(userName) + cpuId +
motherboardSerial`.

Those inputs are stable and observable, so this is **obfuscation against a
determined local attacker, not strong protection**. The interface says so
outright, and falls back to a weak-encryption warning when the hardware
identifiers cannot be read.

Because the key is machine-bound, a profile encrypted on one machine cannot be
decrypted on another. The passphrase-based export and import command in
milestone M6 is the answer to that.

### 3.5 Hardware identity is Windows-only for now

`hw` isolates the platform differences behind one interface. Only Windows is
implemented for now, matching the original's scope:

| Source | CPU identifier | Board identifier |
|--------|----------------|------------------|
| Windows | WMI `Win32_Processor.ProcessorID` | WMI SMBIOS serial, PNP device id as fallback |

When identifiers cannot be read the package reports a weak result so the
user interface can warn that the configuration is only weakly protected.

The lookup sits behind an interface, so adding Linux or macOS later means
adding a platform file and nothing else. The domain code never asks which
platform it is running on.

### 3.6 The database is an embedded document store

Storage is [bbolt](https://go.etcd.io/bbolt), a pure Go embedded key-value
store. One bucket per account and region holds one document per book, with the
components, chapters, series and creators nested inside it. Conversions live in
a sibling bucket because they are updated far more often than books.

```
meta/                             settings, format version
profiles/                         registered device profiles
accounts/<id>-<region>/books/     one document per ASIN
accounts/<id>-<region>/conversions/
```

The original used Entity Framework over fifteen SQL tables with numbered
migrations. Unseal does not. A fresh Go application has no legacy database to
upgrade, so the migration machinery, the backup-before-migrate step and the
evolution cost of the relational schema are all weight without benefit. The
access patterns are document reads and writes, not joins: load a profile's
library, upsert a book by ASIN, update one conversion's state, walk every
conversion during reconciliation.

The tenancy key that the original spread across every query — account plus
region — becomes implicit in the bucket path. The same title may exist
independently for two accounts, which is simpler than sharing a single `Book`
row across accounts and keeping the account-scoped fields separate from the
shared ones.

bbolt is pure Go, so the whole application still builds with `CGO_ENABLED=0`
and ships as a single static binary with no C toolchain anywhere.

### 3.7 The user interface never blocks and never races

The interface is built with [GOUI](https://github.com/golang-gui/goui), a native
desktop GUI toolkit for Go. GOUI loads the platform's native APIs at runtime
through purego, so the application builds with `CGO_ENABLED=0` and needs no C
toolchain on the build machine.

Windows is the only supported platform for now, matching the original.

The interface is declarative. Views are ordinary Go values that may be rebuilt
on every frame, while the framework retains the underlying widgets. Durable
state lives in structures the caller owns, never inside a build closure.

Long-running work — library sync, downloads, decryption — runs off the
interface goroutine. Results travel back through `app.Post` and are applied on
the interface goroutine, which replaces the original's
`AffineSynchronizationContext` with the Go idiom. The domain layer never
mutates an entity that the interface is currently displaying from a background
goroutine.

### 3.8 Progress is increments, not absolutes

A one-hour audiobook reporting every 16 KiB would flood the interface if every
update carried an absolute position. The progress reporter converts a
continuous ratio into increments, remembering the last reported step, and
emits the remainder on close so a bar always reaches 100%.

---

## 4. Data flow

```
   browser  ──sign-in──▶  app  ──▶  audible.Register / Token
                            │
                            ├──▶  db.Migrate          (startup)
                            │
                            ├──▶  audible.Library     ──▶  db upsert books
                            │
   user selects titles
                            │
                            ▼
                        pipeline.Job
                            │
       ┌────────────────────┼────────────────────┐
       ▼                    ▼                    ▼
  audible.License    audible.Download     decrypt.Decryptor
       │                    │                    │
       └────────── state transitions ───────────┘
                            │
                            ▼
                       export.Export
                            │
                            ▼
                   .m4b  +  .aax  +  sidecar JSON
```

---

## 5. Security posture

| Threat | Mitigation |
|--------|------------|
| Credential theft from the local disk | AES-256 encrypted store with an HMAC-signed payload; key derived from the user name plus machine identifiers. |
| Password exposure | The password never reaches the application. Only the authorisation code crosses back from the browser. |
| Secrets in the log | Anonymiser replaces user names and passwords with stable pseudonyms; JSON payloads are written shape-only. |
| Path traversal through titles | Filename pruning strips invalid characters; long paths are normalised to the extended-length form. |
| Tampered downloads | The written size is compared against the size stated in the licence. |
| Malicious update | The update package checksum is verified before the installer is launched. |

Accepted risks: the configuration encryption is obfuscation rather than strong
protection; the update checksum is integrity-only with no code signing;
auto-redirect is deliberately disabled because the sign-in flow relies on the
user copying the final URL.

---

## 6. What is deliberately absent

- No mass distribution of any kind — no sharing, no seeding, no public index.
- No password handling whatsoever.
- No DRM key derivation that bypasses Audible's licence endpoint.
- No content upload paths in the interface or in the protocol client.
