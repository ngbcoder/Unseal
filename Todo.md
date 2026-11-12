# Unseal — Implementation Plan

Living document. Update as work lands. Newest work goes at the top of its section.

Legend: `[ ]` todo · `[~]` in progress · `[x]` done · `[!]` blocked

---

## Overview

**Unseal** is a Go rewrite of *Book Lib Connect* (`audiamus/BookLibConnect`), a
standalone audiobook downloader and DRM remover for a user's own Audible
library. It signs the user in through the browser, registers itself as an
Audible device, lists the purchased library, downloads the encrypted payload,
removes the DRM using the account-scoped keys Audible issued for that account
and book, and writes plain `.m4b` files.

The full behavioural specification lives in [`design.md`](design.md) (the original
C# design document, v0.14.1). This file tracks the Go implementation.

**Legal stance (normative, carried over from the original):** Unseal does not
crack DRM. It performs the same key-provisioning handshake the official
Audible client performs and then uses the account-scoped keys Audible issued
for that specific account and book. It never asks for, stores or transmits the
user's password. Output is for personal archiving only and must not be
uploaded to open servers, torrents or any mass-distribution channel.

---

## Tech stack

| Layer | Technology | Why |
|-------|-----------|-----|
| Language | Go 1.26.4 | Matches the rest of the workspace; single static binary |
| **Constraint** | **`CGO_ENABLED=0` everywhere** | No cgo toolchain needed to build, cross-compilation is trivial, no gcc on the build machine |
| **Target** | **Windows x64** | The only supported platform for now, matching the original. Other platforms are not built, tested or shipped. |
| UI | `github.com/golang-gui/goui` v0.4.0 | Native desktop GUI toolkit. No cgo — the native APIs are loaded at runtime through purego. MIT licensed. |
| Storage | `go.etcd.io/bbolt` v1.5.0 | Pure Go embedded KV store; document-per-book; no ORM, no SQL, no migrations |
| HTTP | stdlib `net/http` | Explicit cookie jar, auto-redirect disabled |
| Crypto | stdlib `crypto/*`, `golang.org/x/crypto` | `crypto/pbkdf2` is stdlib since Go 1.24 |
| HTML→text | `golang.org/x/net/html` | Replaces HtmlAgilityPack |
| Hardware ID | `github.com/stackexchange/wmi` | Windows machine identity for the credential-store key derivation |
| Decryption | ffmpeg `aax` demuxer (`-audible_key`/`-audible_iv`) | Replaces the C# AAXClean package |
| CLI | stdlib `flag` | No dependency until a real need appears |

**Build.** No make, no scripts. The Go toolchain is the build system:

```sh
go build ./...                                    # build everything
go build -o bin/unseal ./cmd/unseal               # build the binary
go test ./...                                    # run the tests
go test -race ./...                              # run the tests with the race detector
go vet ./...                                     # vet
gofmt -l .                                       # formatting check
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...   # release build
```

A release build stamps the version, commit and build date:

```sh
go build -trimpath -ldflags "-s -w \
  -X github.com/ngbcoder/Unseal/internal/version.Version=$(git describe --tags --always) \
  -X github.com/ngbcoder/Unseal/internal/version.Commit=$(git rev-parse --short HEAD) \
  -X github.com/ngbcoder/Unseal/internal/version.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o bin/unseal ./cmd/unseal
```

A plain `go build` needs no ceremony: since Go 1.24 the toolchain embeds the VCS
revision, commit time and dirty-tree flag into the binary, and
`internal/version` recovers them.

**Layers (enforced by import direction, never by a linter):**

| # | Layer | Packages | Rule |
|---|-------|----------|------|
| L0 | Foundation | `logging`, `util`, `hw`, `version` | No product knowledge. Pure utilities. |
| L1 | Shared contracts | `config`, `audible/model` | Types and DTOs shared across layers. |
| L2 | Persistence | `db` | Entities, context, migrations. Knows nothing about HTTP or UI. |
| L3 | Domain | `audible`, `decrypt`, `export`, `pipeline`, `core` | All protocol logic and orchestration. |
| L4 | Presentation | `ui` | GOUI desktop widgets. Depends on L3 through interfaces only. |
| L5 | Composition root | `app`, `cmd/unseal` | Wiring, settings, startup sequence. |

---

## Identity

| Item | Value |
|------|-------|
| Program | Unseal |
| Binary | `unseal.exe` |
| Module | `github.com/ngbcoder/Unseal` |
| Repository | `github.com/ngbcoder/Unseal` (public) |
| License | GPL-3.0 — required, the upstream C# project is GPL-3.0 and this is a derivative |
| Attribution | README credits `audiamus/BookLibConnect` and its author |
| Config root | `%LOCALAPPDATA%\audiamus\Unseal\` |
| Update manifest | `res/Setup.json` in this repository |

---

## Project layout

```
unseal/
├── cmd/unseal/            entry point
├── internal/
│   ├── app/               composition root, startup sequence
│   ├── audible/           protocol client (register, token, library, license, download)
│   │   └── model/         Audible JSON DTOs
│   ├── config/            two-tier settings
│   ├── core/              domain facade
│   ├── db/                bbolt store, document model, repository
│   ├── decrypt/           Decryptor interface + ffmpeg backend
│   ├── export/            .aax + sidecar JSON export
│   ├── hw/                hardware identity per platform
│   ├── logging/           leveled logging facility
│   ├── pipeline/          download/decrypt/export orchestration
│   ├── ui/                GOUI desktop interface
│   ├── util/              pure helpers
│   └── version/           build identity
├── docs/                  architecture, ADRs, user manual
├── res/                   update manifest, images
├── Todo.md
├── URIhandoff.md          the handoff contract with AAX Audio Converter
├── design.md              the original C# design document (reference spec)
└── README.md
```

---

## Deviations from the design document

`design.md` is the specification of **product behaviour and wire protocol**, not
a template for internals. Where the Go implementation departs from it, the
departure is recorded here so it is a decision rather than an accident.

| # | Design document says | Unseal does | Why |
|---|---|---|---|
| 1 | EF Core entities over 15 SQL tables, with migrations | One nested document per book in a bbolt bucket | A fresh Go app has no legacy database to migrate, and the access patterns are document reads and writes, not joins. Removes the ORM, the migration machinery and the composite tenancy key. |
| 2 | `Conversion.AccountId + Region` is the tenancy key | A separate bucket per account and region | Makes the tenancy implicit in the bucket instead of a column pair repeated in every query. The same title may exist independently per account, which is simpler than sharing a `Book` row across accounts. |
| 3 | WinForms user interface | GOUI — native desktop widgets | Gio and Fyne both need cgo for their windowing and GL backends. GOUI loads the native APIs at runtime through purego, so it stays cgo-free. GOUI also ships the widget set this app actually needs — a data table with custom cells, a split view, tabs and tree views — where Gio is immediate-mode and has no data grid, so the library grid would have to be hand-rolled. |
| 4 | `AAXClean` NuGet package decrypts in-process | ffmpeg `aax` demuxer behind a `Decryptor` interface | Go has no AAXClean equivalent. The interface keeps a pure-Go backend possible later. |
| 5 | Configuration encrypted with a hardware-derived key | Same scheme, plus passphrase-based export and import | The original's key is machine-bound, so a profile cannot move between machines. |
| 6 | Windows only | Windows only for now | Same scope as the original. The hardware identity layer sits behind an interface so other platforms can be added later without touching the domain code, but they are not built, tested or shipped yet. |
| 7 | Inno Setup installer and a `Setup.json` updater manifest | GoReleaser-style release workflow and the same manifest | Keeps the update channel identical to the original's. |
| 8 | `EConversionState` ordinal order is semantically meaningful | Preserved exactly | The state machine depends on it. This is behaviour, not structure. |

**Preserved without deviation** because they are protocol or behaviour, not
structure: the ADP signing scheme, the voucher decryption derivation, the
PKCE sign-in flow, the filename derivation rules, the export sidecar formats,
the state machine and its ordinal order, the reconciliation fallback chains,
and the legal stance.

---

## M0 — Foundation · DONE

- [x] `git init` with branch `main` in the project folder
- [x] `core.autocrlf=false`, `core.eol=lf` for consistent line endings
- [x] `.gitignore` — build output, `*.db`, credentials (`Configuration.json`, `*.pem`), editor dirs
- [x] `.gitattributes` — `text=auto eol=lf`, binary rules for media, CRLF for `*.ps1`/`*.bat`/`*.iss`
- [x] `LICENSE` — GPL-3.0, copied verbatim from upstream
- [x] `go.mod` — `module github.com/ngbcoder/Unseal`, `go 1.26.4`
- [x] Build system is the Go toolchain alone — no make, no wrapper scripts
- [x] `README.md` with upstream attribution and the legal notice
- [x] `Todo.md` — this file
- [x] `docs/architecture.md` — layer diagram, dependency rules, key design decisions
- [x] Package skeleton with a `doc.go` per package describing its layer and responsibility
- [x] `internal/version` — link-time injection with a `debug.ReadBuildInfo` fallback
- [x] `internal/version` — recovers the VCS revision, commit time and dirty flag from a plain `go build`
- [x] `internal/version/version_test.go`
- [x] `cmd/unseal/main.go` — entry point, usage banner, `version`/`help`
- [x] `go build ./...`, `go vet ./...`, `go test ./...` all clean
- [x] Verified GOUI v0.4.0, bbolt and `golang.org/x/net/html` compile with `CGO_ENABLED=0` for windows/amd64
- [x] GitHub repository `ngbcoder/Unseal` created (public), `origin` configured
- [x] First push to `origin/main`

---

## M1 — Foundation packages

- [ ] `internal/logging` — level constants `LevelError`..`LevelVerbose` (0..4), `LevelReserved` (5)
- [ ] `internal/logging` — message format `HH:mm:ss.fff TTTT [Context.Method] message`
- [ ] `internal/logging` — level-gated logging (`Log(level, ctx, fn)`) so messages are built lazily
- [ ] `internal/logging` — daily rolling files `<AppName>_<yyyy-MM-dd>_NNN.log`
- [ ] `internal/logging` — roll at 20 MB per file, max 1000 files per day
- [ ] `internal/logging` — `FileModeCreate` semantics, never clobber an existing file
- [ ] `internal/logging` — `InstantFlush` mode plus a 5-second flush timer fallback
- [ ] `internal/logging` — global level settable from `appsettings.json` or `--log <n>`
- [ ] `internal/logging` — `LogGuard` scope helper: logs entry and exit of a method
- [ ] `internal/logging` — tests: level gating, format, daily rollover, flush
- [ ] `internal/paths` — per-OS application data root resolution
- [ ] `internal/paths` — subdirectories `data`, `img`, `config`, `settings`, `log`, `tmp`
- [ ] `internal/paths` — create directories on demand, idempotently
- [ ] `internal/paths` — tests: directory layout is created and reused
- [ ] `internal/config` — `AppSettings` (`LogLevel`, `DbgOnlineUpdate`) read-only tier
- [ ] `internal/config` — `UserSettings` (`Update`, `Download`, `Config`, `Export`) read/write tier
- [ ] `internal/config` — the path of the AAX Audio Converter executable, used by the handoff
- [ ] `internal/config` — JSON marshalling with stable field order and omitempty
- [ ] `internal/config` — atomic write (temp file + rename)
- [ ] `internal/config` — resolution order: user dir → app dir → defaults
- [ ] `internal/config` — change notification so any mutation persists immediately
- [ ] `internal/config` — tests: round-trip, atomicity, fallback order
- [ ] `internal/util` — `Prune` — strip characters invalid in file names
- [ ] `internal/util` — `Prune` — strip the curated "doubtful" typographic set (`¿`, `™`, `—`, smart quotes)
- [ ] `internal/util` — `Prune` — trim trailing dots and spaces
- [ ] `internal/util` — `AsUncIfLong` — `\\?\` extended-length form above 250 chars, Windows only
- [ ] `internal/util` — `RoundDown` — floor a time to a span boundary (10-minute buckets)
- [ ] `internal/util` — `ToXmlTime` — `XmlConvert.ToString(..., Utc)` equivalent
- [ ] `internal/util` — `CRC32` — checksum used for alias uniqueness
- [ ] `internal/util` — `HTMLToText` — strip comments/head/style/script, unwrap span/label, `<br>` to newlines
- [ ] `internal/util` — tests for every helper above, including edge cases
- [ ] `internal/progress` — `Message` type: item count, item increment, per-cent and per-mille increments
- [ ] `internal/progress` — `Reporter` — converts a continuous ratio into increments, remembers the last step
- [ ] `internal/progress` — `Dispose` emits the remainder so a bar always reaches 100%
- [ ] `internal/progress` — tests: increments are monotonic and sum to the total

---

## M2 — Persistence

- [ ] `internal/db` — `Book` document — marketing metadata, delivery type, soft-delete flag
- [ ] `internal/db` — `Component` document — one physical part of a multi-part book
- [ ] `internal/db` — `Conversion` document — the download/decrypt/export unit of work
- [ ] `internal/db` — `ChapterInfo` document — brand intro/outro, runtime, accuracy flag
- [ ] `internal/db` — `Chapter` document — nested via a parent reference
- [ ] `internal/db` — `Series` document — with the three orderings
- [ ] `internal/db` — `Author` and `Narrator` documents — unique ASIN, may lack one
- [ ] `internal/db` — `Genre` document — with the category ladder
- [ ] `internal/db` — `EConversionState` — 16 states, **ordinal order is semantically meaningful**
- [ ] `internal/db` — `ECodec`, `EDeliveryType`, `EDownloadQuality` types
- [ ] `internal/db` — open the bbolt database, create the buckets, enable the file lock
- [ ] `internal/db` — bucket layout: `meta`, `profiles`, `accounts/<id>-<region>/books`, `accounts/<id>-<region>/conversions`
- [ ] `internal/db` — document encoding — one serialisation format, versioned
- [ ] `internal/db` — read a book document by ASIN
- [ ] `internal/db` — upsert a book document, preserving the conversion state of an existing entry
- [ ] `internal/db` — upsert the components of a book
- [ ] `internal/db` — upsert the conversions of a book
- [ ] `internal/db` — list every book of one account and region
- [ ] `internal/db` — list every conversion of one account and region
- [ ] `internal/db` — write the conversion state
- [ ] `internal/db` — write the licence key and IV onto the owning book or component
- [ ] `internal/db` — write the download file name, including the `_Mono`/`_Stereo` suffix
- [ ] `internal/db` — write the exported `.aax` path of a conversion, which the handoff to AAX Audio Converter reads
- [ ] `internal/db` — write the chapter tree
- [ ] `internal/db` — write the series ordering onto the book document
- [ ] `internal/db` — `ApplicableState` — resolve the transient `download` state to the persisted one
- [ ] `internal/db` — `ApplicableDownloadQuality` — minimum quality across components
- [ ] `internal/db` — multi-part book state = minimum of its components' states
- [ ] `internal/db` — soft delete and re-add semantics
- [ ] `internal/db` — pseudo-ASIN allocation for people without one
- [ ] `internal/db` — merge duplicate author entries, preferring the real long ASIN
- [ ] `internal/db` — copy the database file before a destructive change
- [ ] `internal/db` — tests: document round-trip, tenancy isolation, aggregation, chapter tree, state transitions
## M3 — Audible protocol

- [ ] `internal/audible/model` — library response DTOs (product, relationship, series, author)
- [ ] `internal/audible/model` — license response DTOs (content licence, metadata, chapter info)
- [ ] `internal/audible/model` — registration response DTOs (tokens, device info, customer info)
- [ ] `internal/audible/model` — voucher DTO (key, IV, rules)
- [ ] `internal/audible/model` — catalogue product DTO
- [ ] `internal/audible/locale` — region → TLD and marketplace-id table (11 regions)
- [ ] `internal/audible/locale` — pre-Amazon login path allowed for de, uk, us only
- [ ] `internal/audible/login` — device serial: 40 chars from `A–Z a–z 0–9`
- [ ] `internal/audible/login` — client id: hex encoding of `<serial>#A2CZJZGLK2JJVM`
- [ ] `internal/audible/login` — PKCE verifier: 32 random bytes, URL-safe base64
- [ ] `internal/audible/login` — PKCE challenge: `BASE64URL(SHA256(verifier))`, method `S256`
- [ ] `internal/audible/login` — sign-in URL builder with the full OpenID Connect 2.0 query set
- [ ] `internal/audible/login` — parse the redirected URL into an authorisation code
- [ ] `internal/audible/sign` — ADP signature: `RoundDown(10 min)` → XML time
- [ ] `internal/audible/sign` — signed data string `METHOD\nurl\ntime\nbody\nadpToken`
- [ ] `internal/audible/sign` — SHA-256 digest, RSA PKCS#1 v1.5 signature over the digest
- [ ] `internal/audible/sign` — signature header value `base64(sig):<time>`
- [ ] `internal/audible/sign` — import the device private key from PEM
- [ ] `internal/audible/sign` — headers `x-adp-token`, `x-adp-alg`, `x-adp-signature`
- [ ] `internal/audible/sign` — tests: fixed vectors for the signed data string and signature
- [ ] `internal/audible/client` — HTTP client with an explicit cookie jar
- [ ] `internal/audible/client` — auto-redirect disabled
- [ ] `internal/audible/client` — gzip/deflate decompression
- [ ] `internal/audible/client` — per-host base addresses for `api.amazon.<tld>` and `api.audible.<tld>`
- [ ] `internal/audible/register` — `POST /auth/register` with the documented request body
- [ ] `internal/audible/register` — parse device info, customer info, bearer token, mac_dms token
- [ ] `internal/audible/register` — extract the device private key (PEM) and the ADP token
- [ ] `internal/audible/register` — extract website cookies and the store-authentication cookie
- [ ] `internal/audible/token` — `POST /auth/token` refresh, form-encoded
- [ ] `internal/audible/token` — header `x-amzn-identity-auth-domain`
- [ ] `internal/audible/token` — skip the refresh when the token is valid for more than 5 minutes
- [ ] `internal/audible/library` — `GET /1.0/library` with paging until an empty page
- [ ] `internal/audible/library` — `purchased_after` from the newest purchase date plus 1 ms
- [ ] `internal/audible/library` — `num_results=100`, `page`, fixed `response_groups` list
- [ ] `internal/audible/library` — de-duplicate by ASIN, sort by purchase date
- [ ] `internal/audible/catalog` — `GET /1.0/catalog/products/{asin}` back-fill
- [ ] `internal/audible/license` — `POST /1.0/content/{asin}/licenserequest`
- [ ] `internal/audible/license` — handle `Granted` and `Denied`, log the denial reasons
- [ ] `internal/audible/license` — **voucher decryption** — `SHA256(ASCII(deviceType+serial+accountId+asin))`
- [ ] `internal/audible/license` — voucher decryption — key and IV are the two halves of the digest
- [ ] `internal/audible/license` — voucher decryption — AES-256-CBC, no padding, truncate at the first 0x00
- [ ] `internal/audible/license` — voucher decryption — tests with a fixed vector
- [ ] `internal/audible/license` — extract the offline URL, content size and content format
- [ ] `internal/audible/license` — extract the chapter tree, brand intro/outro and runtime
- [ ] `internal/audible/download` — `GET offline_url` with the Audible user agent
- [ ] `internal/audible/download` — stream to `.aaxc` with progress reported in absolute bytes
- [ ] `internal/audible/download` — validate the written size against `content_size_in_bytes`
- [ ] `internal/audible/download` — cancellation checked per buffer chunk

---

## M4 — Decrypt and export

- [ ] `internal/decrypt` — `Decryptor` interface: input, output, key, IV, progress, cancellation
- [ ] `internal/decrypt/ffmpeg` — locate ffmpeg: `UNSEAL_FFMPEG` env → `PATH` → bundled directory
- [ ] `internal/decrypt/ffmpeg` — invoke with `-audible_key <hex>` and `-audible_iv <hex>`
- [ ] `internal/decrypt/ffmpeg` — `-i in.aaxc -c copy out.m4b`
- [ ] `internal/decrypt/ffmpeg` — parse progress from stderr
- [ ] `internal/decrypt/ffmpeg` — propagate cancellation as a process kill
- [ ] `internal/decrypt/ffmpeg` — surface a clear error when ffmpeg is missing
- [ ] `internal/decrypt` — report the audio channel count after conversion
- [ ] `internal/decrypt` — append the `_Mono` / `_Stereo` suffix and persist it
- [ ] `internal/export` — copy `.m4b` to `<exportDir>\<name>.aax`
- [ ] `internal/export` — record the exported `.aax` path on the conversion, so the handoff can find it
- [ ] `internal/export` — cancellable copy with per-mille progress
- [ ] `internal/export` — serialise the copy so concurrent parts cannot collide
- [ ] `internal/export` — write `content_metadata_<ASIN>.json` — chapter tree, runtimes, brand intro/outro
- [ ] `internal/export` — write `<ASIN>.json` — compact product record
- [ ] `internal/export` — write `series_titles_<SERIES_ASIN>.json` — ordered book list
- [ ] `internal/export` — series ordering preference: `Sort` → `BookNumber`/`SubNumber` → `Sequence`
- [ ] `internal/export` — write the series sidecar only once for a multi-part book
- [ ] `internal/export` — drop a leading chapter that repeats the brand intro across parts
- [ ] `internal/export` — chapter de-dup rule: same title, length within 1500 ms, shorter than 25 s
- [ ] `internal/export` — filename derivation: prune the title, truncate to 20 characters
- [ ] `internal/export` — filename derivation: `_<ASIN>_LC` plus optional `_<bitrate>_<samplerate>`
- [ ] `internal/export` — tests: sidecar JSON shape, chapter skip rule, filename derivation, series ordering

---

## M9 — Handoff to AAX Audio Converter

Build order: this milestone sits after M4, because it needs the export stage
and the persisted exported path. It comes before M5, so the end-to-end path
from download to conversion is available before the pipeline and the
interface are finished.

Implements the sender side of the contract in [`URIhandoff.md`](URIhandoff.md).
The receiver is the separately developed `github.com/ngbcoder/AAXAudioConverter`.

- [ ] `internal/handoff` — the handoff document — protocol, source, created, files, options
- [ ] `internal/handoff` — the file entry — path, asin, title, authors, narrators, series, seriesSequence, coverPath, durationMs
- [ ] `internal/handoff` — build the document from a set of conversions
- [ ] `internal/handoff` — write the document to the temp directory as `handoff-<token>.json`
- [ ] `internal/handoff` — the token is 8 lowercase hex characters, unique per handoff
- [ ] `internal/handoff` — write the document through a temporary sibling, then rename, so the receiver never sees a partial file
- [ ] `internal/handoff` — the document protocol version is 1
- [ ] `internal/handoff` — the options are omitted, so the receiver keeps its own settings
- [ ] `internal/handoff` — build the URI `aaxconv://add?doc=<encoded>&activate=1&src=unseal`
- [ ] `internal/handoff` — percent-encode with `url.Values`, never by string concatenation
- [ ] `internal/handoff` — the URI stays well under the Windows command-line limit
- [ ] `internal/handoff` — spawn the receiver executable with the URI as `argv[1]`
- [ ] `internal/handoff` — hide the console window on Windows, no console flash
- [ ] `internal/handoff` — detect an immediate spawn failure and report it
- [ ] `internal/handoff` — do not wait for the receiver and do not start a conversion
- [ ] `internal/handoff` — the sender's own state machine does not change, a book stays `exported`
- [ ] `internal/handoff` — locate the receiver — the uninstall registry key first, then the user setting
- [ ] `internal/handoff` — read `HKLM` and `HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\{AAX Audio Converter}_is1\InstallLocation`
- [ ] `internal/handoff` — the receiver executable is `AaxAudioConverter.exe` in that directory
- [ ] `internal/handoff` — the user setting holds the path and the user can browse for it
- [ ] `internal/handoff` — the button — "Convert with AAX Audio Converter", in the download-queue toolbar
- [ ] `internal/handoff` — the button is enabled only when a row is selected
- [ ] `internal/handoff` — the button is enabled only when every selected row is `exported` or `converted`
- [ ] `internal/handoff` — the button is enabled only when the converter path is configured and the file exists
- [ ] `internal/handoff` — the tooltip states which condition failed
- [ ] `internal/handoff` — rows may belong to different books, all of them go in one request
- [ ] `internal/handoff` — a multi-part book sends one file per part, the receiver groups them itself
- [ ] `internal/handoff` — `testdata/handoff/minimal.json` — one file, no metadata, no options
- [ ] `internal/handoff` — `testdata/handoff/rich.json` — two files of one multi-part book, full metadata, options
- [ ] `internal/handoff` — `testdata/handoff/empty.json` — the degenerate case, `files: []`
- [ ] `internal/handoff` — `testdata/handoff/uri_minimal.txt` and `uri_doc.txt`
- [ ] `internal/handoff` — the test asserts the built document matches the golden fixture
- [ ] `internal/handoff` — the test parses the golden URIs and asserts the verb and the parameters
- [ ] `internal/handoff` — the golden fixtures are the same files the receiver repository uses

## M5 — Pipeline

- [ ] `internal/pipeline` — job driver over a set of conversions
- [ ] `internal/pipeline` — stage order: licence → download → decrypt → export
- [ ] `internal/pipeline` — persist the conversion state after every transition
- [ ] `internal/pipeline` — decide per conversion whether a download is needed from the persisted state
- [ ] `internal/pipeline` — decide per conversion whether a decryption is needed
- [ ] `internal/pipeline` — a higher requested quality forces download and decrypt
- [ ] `internal/pipeline` — decryption semaphore, default 1, configurable
- [ ] `internal/pipeline` — downloads serialised per book, decryption parallelised
- [ ] `internal/pipeline` — cancellation checked before each book's licence step
- [ ] `internal/pipeline` — export only once every part of a book is unlocked
- [ ] `internal/pipeline` — download progress from byte counts
- [ ] `internal/pipeline` — decrypt progress from media position
- [ ] `internal/pipeline` — export progress from copy progress
- [ ] `internal/pipeline` — reconciliation — walk every conversion and check the file on disk
- [ ] `internal/pipeline` — reconciliation — `local_locked` → `.aaxc` → `remote`
- [ ] `internal/pipeline` — reconciliation — `local_unlocked` → `.m4b` → `.aaxc` → `remote`
- [ ] `internal/pipeline` — reconciliation — `exported` → `.aax` → download dir → `converted_unknown`
- [ ] `internal/pipeline` — reconciliation — `converted` → any audio artefact in the destination
- [ ] `internal/pipeline` — reconciliation — relocated files: find the file elsewhere and rewrite the path
- [ ] `internal/pipeline` — reconciliation — missing payload: clear the stored file name
- [ ] `internal/pipeline` — reconciliation — ask the user once, then apply to all
- [ ] `internal/pipeline` — tests: state transitions, quality upgrade, reconciliation table

---

## M6 — Security

- [ ] `internal/secure` — AES-256 with a PBKDF2-HMAC-SHA256 derived key, 100 000 iterations
- [ ] `internal/secure` — separate authentication and key salts
- [ ] `internal/secure` — random IV per encryption
- [ ] `internal/secure` — HMAC-SHA256 tag verified in constant time
- [ ] `internal/secure` — payload layout `[authSalt 16][keySalt 16][iv 16][ciphertext][hmac 32]`
- [ ] `internal/secure` — tests: round-trip, tamper detection, wrong key, malformed input
- [ ] `internal/hw` — Windows CPU id via WMI `Win32_Processor.ProcessorID`
- [ ] `internal/hw` — Windows motherboard serial via WMI SMBIOS, PNP device id as fallback
- [ ] `internal/hw` — the identifier lookup sits behind an interface so other platforms can be added later
- [ ] `internal/hw` — report a weak result when identifiers are unavailable
- [ ] `internal/secure` — key derivation: `ROT13(userName) + cpuId + motherboardSerial`
- [ ] `internal/secure` — configuration store — `Configuration.json` with a plaintext `Profiles` field
- [ ] `internal/secure` — configuration store — encrypted `Secure` field, base64 payload
- [ ] `internal/secure` — configuration store — read path: try without a token, force the token if encrypted
- [ ] `internal/secure` — passphrase-based profile export and import for cross-machine migration
- [ ] `internal/logging` — anonymiser — stable pseudonyms for usernames and passwords
- [ ] `internal/logging` — replace the user's profile path with a placeholder
- [ ] `internal/logging` — HTTP request and response dumps at level 4 and above
- [ ] `internal/logging` — JSON payloads written shape-only, all strings replaced by length placeholders

---

## M7 — Command line and desktop interface

- [ ] `cmd/unseal` — command tree with `run`, `register`, `library`, `download`, `export`, `config`, `profiles`, `doctor`, `version`
- [ ] `cmd/unseal` — global flags: `--log <n>`, `--config <path>`, `--profile <alias>`
- [ ] `cmd/unseal` — `unseal doctor` — report ffmpeg availability, config paths, database health
- [ ] `internal/ui` — `ui.Run` entry point from `main`, never from a goroutine
- [ ] `internal/ui` — application id `com.audiamus.unseal`, stable across releases
- [ ] `internal/ui` — Modern theme via `modern.FromSettings(app.Settings())`
- [ ] `internal/ui` — design tokens — colours, spacing, typography as style values
- [ ] `internal/ui` — main window — settings panels for profile, download and export
- [ ] `internal/ui` — main window — download queue table, one row per conversion
- [ ] `internal/ui` — main window — three progress bars: download, decrypt, export
- [ ] `internal/ui` — main window — Run and Abort buttons
- [ ] `internal/ui` — main window — Run enabled only when a selection and the directories exist
- [ ] `internal/ui` — library window — table with state, title, author, duration, year, narrator, series, genre, parts
- [ ] `internal/ui` — library window — virtualised list so a large library stays responsive
- [ ] `internal/ui` — library window — filter by adult products and hide unavailable
- [ ] `internal/ui` — library window — sort by the configured initial sorting
- [ ] `internal/ui` — library window — rows in the `unknown` state are not selectable
- [ ] `internal/ui` — library window — split view with a read-only metadata panel
- [ ] `internal/ui` — library window — cover image via `icon.NewImage` from the cached file
- [ ] `internal/ui` — library window — publisher summary rendered as plain text
- [ ] `internal/ui` — download staging — add selected, remove selected, remove all
- [ ] `internal/ui` — download staging — staged rows highlighted, state icon in the first column
- [ ] `internal/ui` — download staging — one row per component for multi-part books
- [ ] `internal/ui` — stable row identifiers so sorting never loses the selection
- [ ] `internal/ui` — profile dialogs — manage, rename, remove
- [ ] `internal/ui` — profile dialogs — new registration: create URL, open browser, paste redirect
- [ ] `internal/ui` — profile dialogs — account alias, minimum 3 characters, unique by CRC32
- [ ] `internal/ui` — first-run wizard — profile, download directory, export decision, help
- [ ] `internal/ui` — file dialogs for the download and export directories
- [ ] `internal/ui` — background work results marshalled back with `app.Post`
- [ ] `internal/ui` — plain field changes trigger `app.RequestUpdate`
- [ ] `internal/ui` — progress rendering from the progress stream
- [ ] `internal/ui` — tests: HTML to text golden cases, state icon mapping, alias uniqueness
## M8 — Operations and release

- [ ] `internal/update` — fetch the manifest from `res/Setup.json` in this repository
- [ ] `internal/update` — compare the published version against the running version
- [ ] `internal/update` — download the installer to the user's download directory
- [ ] `internal/update` — verify the checksum before launching
- [ ] `internal/update` — prompt before download and again before install
- [ ] `internal/update` — defer the install when the application is busy
- [ ] `internal/diag` — write library, licence and product responses to the temp directory
- [ ] `internal/diag` — write a settings dump using the object-graph printer
- [ ] `internal/logging` — housekeeping — prune the temp and log directories
- [ ] `internal/logging` — housekeeping — thresholds: 500 files, 100 MB, 365 days
- [ ] `internal/logging` — housekeeping — cross-enforce between the temp and log directories
- [ ] `internal/i18n` — resource strings for English and German
- [ ] `internal/i18n` — region names, state descriptions and captions
- [ ] `.github/workflows/ci.yml` — format check, vet, race tests
- [ ] `.github/workflows/ci.yml` — the race detector needs cgo, so the runner needs a C compiler. GitHub's windows runners ship mingw
- [ ] `.github/workflows/ci.yml` — build matrix: windows/amd64
- [ ] `.github/workflows/release.yml` — build, checksum, publish a GitHub release
- [ ] `.github/workflows/release.yml` — update `res/Setup.json` with the new version
- [ ] `docs/` — user manual, architecture document, ADRs
- [ ] `README.md` — screenshots, usage, legal notice, upstream attribution

---

## Known gaps

- [ ] **ffmpeg is an external runtime dependency.** The C# original had none
      (AAXClean was in-process). Unseal needs a bootstrap downloader so that
      ffmpeg is never a manual prerequisite.
- [ ] **Credentials are bound to the machine.** The configuration key derives
      from machine identifiers, so a profile encrypted on one machine cannot
      be decrypted on another. The export/import command in M6 addresses this.
- [ ] **Only Windows is supported.** The code avoids OS-specific calls outside
      `internal/hw` and `internal/ui`, so other platforms can be added later,
      but they are not built, tested or shipped yet.
- [ ] **GOUI v0.4.0 is young.** It was published on 2026-10-26 and its API is
      still moving, so the interface layer will need occasional adjustment.
- [ ] **Audible's protocol can change server-side without notice.** The Go
      ecosystem has far fewer reference implementations than C#, so each
      protocol change has to be derived from captured traffic.
- [ ] **The user interface can only be tested by hand.** The pipeline and
      crypto layers get real unit tests; the GOUI layer gets golden-image and
      model tests.

---

## Backlog (post-v1)

- [ ] Pure-Go AAX/MP4 decryptor, so ffmpeg is not needed at all
- [ ] MP3 export format alongside the `.aax` interchange format
- [ ] Cover image cache with a bounded size and prefetch
- [ ] Light theme alongside the dark theme
- [ ] Multiple accounts side by side in one window
- [ ] Resume a partially downloaded payload with an HTTP range request
- [ ] Export a library index as CSV or JSON
