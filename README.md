# Unseal

A standalone audiobook downloader and DRM remover for your **own** Audible
library, written in Go.

Unseal signs you in through your own browser, registers itself as an Audible
device, lists the titles you purchased, downloads the encrypted payload,
removes the DRM using the account-scoped keys Audible issued for your account
and that specific book, and writes plain `.m4b` files you can play anywhere.

It can additionally re-package the result into the `.aax` + sidecar-JSON layout
that [AAX Audio Converter](https://github.com/audiamus/AaxAudioConverter)
consumes.

> **Origin.** Unseal is an independent Go rewrite of
> [Book Lib Connect](https://github.com/audiamus/BookLibConnect) by
> [audiamus](https://github.com/audiamus), which is licensed under the GPL-3.0.
> Unseal is distributed under the same licence and exists thanks to that
> original work.

---

## Legal and ethical stance

This section is normative.

- Unseal **does not crack DRM**. It performs the *same* key-provisioning
  handshake the official Audible client performs, then uses the account-scoped
  keys Audible issued for that specific account and book.
- Unseal **never asks for, stores, or transmits your password.** Sign-in
  happens entirely in your browser; only the authorisation code crosses back.
- Output is for your **personal** archiving, conversion and convenience only.
- De-DRM'd files **must not** be uploaded to open servers, torrents, or any
  mass-distribution channel. No support is offered for such use.
- There is deliberately no "share library", "upload", or "public index"
  feature. Every API call either reads your own library, requests a licence for
  a title you own, or moves bytes you already have.

---

## Status

Under active development. The project follows a milestone plan that is tracked
in [`Todo.md`](Todo.md).

| Milestone | Contents | State |
|-----------|----------|-------|
| M0 | Repository, build system, package skeleton | done |
| M1 | Logging, settings, paths, helpers, progress | planned |
| M2 | Persistence — entities, migrations, repository | planned |
| M3 | Audible protocol — registration, library, licence, download | planned |
| M4 | Decryption and export | planned |
| M5 | Pipeline, state machine, reconciliation | planned |
| M6 | Credential store encryption, hardware identity | planned |
| M7 | Command line and GOUI desktop interface | planned |
| M8 | Self-update, diagnostics, CI and release | planned |
| M9 | Handoff to AAX Audio Converter, per [`URIhandoff.md`](URIhandoff.md) | planned |

---

## Requirements

- **Windows x64** — the only supported platform for now
- Go 1.26.4 or newer (to build)
- [ffmpeg](https://ffmpeg.org/) with the `aax` demuxer (to decrypt)

ffmpeg is located at runtime in this order:

1. the path in the `UNSEAL_FFMPEG` environment variable
2. `ffmpeg` on `PATH`
3. an `ffmpeg` binary in the directory next to the Unseal executable

Run `unseal doctor` to check.

---

## Building

The Go toolchain is the build system. There is no make and no wrapper script.

```sh
go build ./...                  # build everything
go build -o bin/unseal ./cmd/unseal
go test ./...                  # run the tests
go test -race ./...            # run the tests with the race detector
go vet ./...                   # vet
gofmt -l .                     # formatting check
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

The whole tree builds with `CGO_ENABLED=0`, so no C toolchain is needed:

```sh
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...
```

---

## Using

```sh
unseal doctor        # check the environment and dependencies
unseal register      # register a new Audible device profile
unseal library       # list the library of the selected profile
unseal download      # download and decrypt selected titles
unseal export        # export decrypted titles to .aax plus sidecars
unseal run           # start the desktop application
```

The desktop interface is the primary experience; the command line exists for
scripting and for diagnostics.

---

## Testing

```sh
go test ./...                   # the whole suite
go test -coverprofile=coverage.out ./...
```

The race detector needs cgo, so it needs a C compiler. This machine has two:

```sh
# MSYS2, already present
export PATH="/c/msys64/mingw64/bin:$PATH"

# or WinLibs, installed through winget
export PATH="$LOCALAPPDATA/Microsoft/WinGet/Packages/BrechtSanders.WinLibs.POSIX.UCRT_Microsoft.Winget.Source_8wekyb3d8bbwe/mingw64/bin:$PATH"

export CGO_ENABLED=1
export CC=gcc
go test -race ./...
```

The release build stays `CGO_ENABLED=0`; only the tests need a compiler.

---

## Architecture

The application is layered so that the protocol and orchestration code never
depends on the user interface.

| Layer | Packages | Rule |
|-------|----------|------|
| L0 | `logging`, `util`, `hw`, `version` | Foundation. No product knowledge. |
| L1 | `config`, `audible/model` | Shared contracts and data transfer objects. |
| L2 | `db` | Persistence. Knows nothing about HTTP or the UI. |
| L3 | `audible`, `decrypt`, `export`, `pipeline`, `core` | Domain and application logic. |
| L4 | `ui` | GOUI desktop widgets. Depends on L3 through interfaces only. |
| L5 | `app`, `cmd/unseal` | Composition root and entry point. |

See [`docs/architecture.md`](docs/architecture.md) for the detail.

---

## Licence

GNU General Public License v3.0. See [`LICENSE`](LICENSE).
