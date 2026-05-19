# Building & Deploying Eiyaro

## Overview

Eiyaro is a high-throughput Proof-of-Work blockchain full node written in Go. This document covers how to build Eiyaro from source on Windows, Linux, and macOS, including Docker-based deployment and the CI/CD pipeline powered by GitHub Actions.

The source tree produces several binaries:

| Binary | Entry Point | Purpose |
|---|---|---|
| `eyarod` | `./cmd/eyarod` | Full node daemon |
| `eiyaroctl` | `./cmd/eiyaroctl` | CLI control tool |
| `eiyarowallet` | `./cmd/eiyarowallet` | Wallet daemon |
| `miner` | `./cmd/miner` | Standalone miner |
| `genkeypair` | `./cmd/genkeypair` | Key-pair generator |
| `getheight` | `./cmd/getheight` | Utility to query chain height |
| `ldbtool` | `./cmd/ldbtool` | LevelDB inspection utility |

---

## Prerequisites

### Go Version

The module requires **Go 1.26.0** (declared in `go.mod` as `go 1.26.0`). The project module path is:

```
github.com/Hoosat-Oy/HTND
```

CI jobs use Go 1.25.7 for testing and Go 1.26 for release builds.

### Supported Platforms

| Platform | Architecture | Status |
|---|---|---|
| Linux (Ubuntu 24.04) | amd64 | Primary target |
| Windows 10/11 | amd64 | Fully supported |
| macOS | amd64 | Supported (release artifacts) |

### Required System Packages (Linux)

```bash
apt-get update && apt-get install -y curl git openssh-client binutils gcc musl-dev
```

These are installed automatically in the Docker build stage.

---

## Build Scripts

### Windows (build.bat)

`build.bat` supports three modes selected by an argument:

```cmd
REM Production build (default) — stripped binary
build.bat

REM Build with race detector
build.bat race

REM Development build — no stripping, faster compilation
build.bat dev
```

All output goes to `bin\eyarod.exe`. The script aborts on any build failure.

Build flags in production mode:
- `-ldflags="-s -w"` — strip debug symbols, reduce binary size

Build flags in race mode:
- `-race` — enable the Go race detector
- `-vet` — run go vet during compilation
- `-ldflags="-s -w"` — strip symbols

Output binary: `bin\eyarod.exe` (or `bin\eyarod-race.exe` in race mode).

### Linux/macOS (build.sh)

`build.sh` accepts an optional argument:

| Command | Effect |
|---|---|
| `./build.sh` | Build `eyarod` (production, stripped) |
| `./build.sh race` | Build `eyarod` with race detector |
| `./build.sh dev` | Build `eyarod` (development, no stripping) |
| `./build.sh all` | Build **all** binaries: `eyarod`, `eiyaroctl`, `eiyarowallet`, `miner`, `genkeypair` |

All output goes to `bin/`. The script uses `set -e` to stop on the first error.

---

## Go Modules

### Key Dependencies

The `go.mod` specifies these primary dependencies:

```
go 1.26.0

require (
    github.com/btcsuite/btcutil v1.0.2
    github.com/btcsuite/go-socks v0.0.0-20170105172521-4720035b7bfd
    github.com/btcsuite/winsvc v1.0.0
    github.com/cespare/xxhash/v2 v2.3.0
    github.com/chewxy/math32 v1.11.1
    github.com/cockroachdb/errors v1.12.0
    github.com/cockroachdb/pebble/v2 v2.1.4
    github.com/davecgh/go-spew v1.1.1
    github.com/gofrs/flock v0.13.0
    github.com/jessevdk/go-flags v1.6.1
    github.com/jrick/logrotate v1.1.2
    github.com/kaspanet/go-muhash v0.0.4
    github.com/kaspanet/go-secp256k1 v0.0.7
    github.com/pkg/errors v0.9.1
    github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10
    github.com/syndtr/goleveldb v1.0.0
    github.com/tyler-smith/go-bip39 v1.1.0
    golang.org/x/crypto v0.49.0
    golang.org/x/sys v0.42.0
    golang.org/x/term v0.41.0
    google.golang.org/grpc v1.79.3
    google.golang.org/protobuf v1.36.11
    lukechampine.com/blake3 v1.4.1
)
```

### Updating Dependencies

```bash
go mod download   # fetch all dependencies
go mod tidy       # prune unused and add missing modules
```

---

## Makefile Targets

The `Makefile` provides the full build pipeline. All targets are listed in the `.PHONY` declaration.

### Build Targets

**`make build`** — Production build (default).

```bash
go build -ldflags="-s -w" -o bin/eyarod ./cmd/eyarod
```

**`make build-race`** — Build with race detector.

```bash
go build -race -vet -ldflags="-s -w" -o bin/eyarod-race ./cmd/eyarod
```

**`make production`** — Alias for `build`. Outputs `bin/eyarod`.

### Testing Targets

**`make test`** — Run all tests (no race detector, 20-minute timeout).

```bash
go test -timeout 20m ./...
```

**`make test-race`** — Run all tests with race detector (20-minute timeout).

```bash
go test -race -timeout 20m ./...
```

**`make test-coverage`** — Run tests with race detector and generate an HTML coverage report.

```bash
go test -race -timeout 20m -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

The HTML report is written to `coverage.html`.

### Code Quality Targets

**`make fmt`** — Format all Go source files in-place using `gofmt`.

```bash
gofmt -w -s .
```

**`make fmt-check`** — Check formatting without modifying files. Returns exit code 1 if any file is not properly formatted. Used in CI.

```bash
gofmt -l . | grep -v vendor | head -1
```

**`make vet`** — Run `go vet` across all packages.

```bash
go vet ./...
```

**`make lint`** — Install `staticcheck` and run it with a curated set of checks. The checks include categories SA4006–SA4023, SA5000–SA5012, SA6001–SA6002, SA9001–SA9006, and ST1019 covering:
- Unused code and variables
- Incorrect error handling patterns
- Suspicious constructs and type assertions
- Performance issues
- Style violations

```bash
go install honnef.co/go/tools/cmd/staticcheck@latest
staticcheck -checks SA4006,SA4008,SA4009,... ./...
```

### Dependency Targets

**`make deps`** — Download all module dependencies.

```bash
go mod download
```

**`make tidy`** — Clean up `go.mod` and `go.sum`.

```bash
go mod tidy
```

### Other Targets

**`make install`** — Install `eyarod` into `$GOPATH/bin` (stripped).

```bash
go install -ldflags="-s -w" ./cmd/eyarod
```

**`make clean`** — Remove `bin/` directory and coverage artifacts.

```bash
rm -rf bin
rm -f coverage.out coverage.html
```

**`make ci`** — Full CI pipeline. Runs sequentially:

1. `deps` — download dependencies
2. `fmt-check` — verify formatting
3. `vet` — go vet
4. `lint` — staticcheck
5. `test-race` — tests with race detector
6. `build-race` — build with race detector

**`make help`** — Display all targets and usage examples.

---

## Docker Build

### Multi-Stage Dockerfile

The [Dockerfile](../Dockerfile) uses a two-stage build:

**Stage 1 — Build** (`golang:1.26`):

```dockerfile
FROM golang:1.26 AS build
ENV GOEXPERIMENT=simd,jsonv2
```

Go experiments enabled:
- `simd` — SIMD optimizations for Hoohash proof-of-work
- `jsonv2` — experimental JSON v2 implementation

Build-time system packages: `curl`, `git`, `openssh-client`, `binutils`, `gcc`, `musl-dev`.

The five binaries are built with build tags:

```dockerfile
RUN go build -tags "deadlock pebblegozstd" -o eyarod .
RUN go build -tags "deadlock pebblegozstd" -o eiyarowallet ./cmd/eiyarowallet
RUN go build -tags "deadlock pebblegozstd" -o eiyarominer ./cmd/miner
RUN go build -tags "deadlock pebblegozstd" -o eiyaroctl ./cmd/eiyaroctl
RUN go build -tags "deadlock pebblegozstd" -o genkeypair ./cmd/genkeypair
```

Build tags:
- `deadlock` — enables mutex deadlock detection instrumentation
- `pebblegozstd` — enables Zstandard compression for the Pebble storage engine

**Stage 2 — Runtime** (`ubuntu:24.04`):

```dockerfile
FROM ubuntu:24.04
WORKDIR /app
```

Runtime packages: `ca-certificates` only. All build tools are discarded.

The container creates a data directory with restricted permissions:

```dockerfile
RUN mkdir -p /nonexistent/.eiyarod && chown nobody:nogroup /nonexistent/.eiyarod && chmod 700 /nonexistent/.eiyarod
```

All binaries are owned by `nobody:nogroup` and run under the `nobody` user for security.

### Building the Docker Image

```bash
docker build -t eiyaro:latest .
```

### Running the Docker Container

```bash
docker run -d \
  -v /path/to/data:/nonexistent/.eiyarod \
  -p 16110:16110 \
  -p 16111:16111 \
  eiyaro:latest
```

### Default Entrypoint

```dockerfile
ENTRYPOINT ["/app/eyarod"]
CMD ["--utxoindex", "--saferpc"]
```

The node starts with:
- `--utxoindex` — enables the UTXO index for address-based queries
- `--saferpc` — enables safe RPC mode for public-facing nodes

Override the command as needed:

```bash
docker run eiyaro:latest --connect=seed.example.com:16111
```

### Exposed Ports (Application Layer)

The default ports defined in the node configuration:
- **16110** — RPC API
- **16111** — P2P networking

---

## Cross-Compilation

### Windows (from Linux)

```bash
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go build -ldflags="-s -w" -o bin/eyarod.exe ./cmd/eyarod
```

Set `CGO_ENABLED=0` to produce a pure-Go statically linked binary. The Windows binary does not require any C runtime.

### Linux (static linking)

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
  go build -ldflags="-s -w" -o bin/eyarod ./cmd/eyarod
```

For the release pipeline, Linux builds use additional linker flags for fully static binaries:

```bash
go build -v -ldflags="-s -w -extldflags=-static" -tags netgo,osusergo -o ./bin/ . ./cmd/...
```

- `-extldflags=-static` — produce a fully static binary (no dynamic library dependencies)
- `-tags netgo` — use the pure-Go network stack (not cgo-based)
- `-tags osusergo` — use pure-Go user lookup (not cgo-based)

### macOS

```bash
GOOS=darwin GOARCH=amd64 \
  go build -ldflags="-s -w" -o bin/eyarod ./cmd/eyarod
```

The release pipeline uses:

```bash
go build -v -ldflags="-s -w" -o ./bin/ . ./cmd/...
```

### ARM64

For ARM64 Linux builds:

```bash
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 \
  go build -ldflags="-s -w" -o bin/eyarod ./cmd/eyarod
```

---

## CI/CD Pipeline

All workflows use [actions/checkout@v4](https://github.com/actions/checkout) and [actions/setup-go@v5](https://github.com/actions/setup-go).

### Tests Workflow (`tests.yaml`)

**Trigger:** Every `push` and `pull_request` (on `opened`, `synchronize`, `edited`).

**Jobs:**

#### 1. Build & Test Matrix

| OS | Runner | Go Version | GOFLAGS |
|---|---|---|---|
| ubuntu-latest | Linux | 1.25.7 | `-tags=ci` |
| windows-latest | Windows | 1.25.7 | `-tags=ci` |

Both platforms run `./build_and_test.sh -v`.

Windows-specific steps:
- Disables CRLF conversion: `git config --global core.autocrlf false`
- Increases the Windows pagefile size via `SetPageFileSize.ps1` to prevent out-of-memory errors

#### 2. Fast Stability Tests

Runs on `ubuntu-latest` with Go 1.25.7.

```bash
go install ./...
cd stability-tests
SKIP_LONG_STABILITY_TESTS=1 ./install_and_test.sh
```

#### 3. Code Coverage

Runs on `ubuntu-latest` with Go 1.25.7.

```bash
go test -v -covermode=atomic \
  -coverpkg "$coverpkg" \
  -coverprofile coverage.txt \
  $packages
```

The coverage is uploaded to [Codecov](https://codecov.io) using `codecov/codecov-action@v5`. The upload step is skipped if the `CODECOV_TOKEN` secret is not configured.

### Deploy Workflow (`deploy.yaml`)

**Trigger:** On `release` `published`.

**Build Matrix:**

| OS | Runner | Go Version | Output Archive |
|---|---|---|---|
| Linux | ubuntu-latest | 1.26 | `Eiyaro-{tag}-linux-amd64.zip` |
| Windows | windows-latest | 1.26 | `Eiyaro-{tag}-windows-amd64.zip` |
| macOS | macos-latest | 1.26 | `Eiyaro-{tag}-osx.zip` |

All builds set `GOEXPERIMENT=simd,jsonv2`.

Linux adds `-extldflags=-static` and `-tags netgo,osusergo` for fully static binaries.

Each archive includes all binaries from `./bin/` and is accompanied by a SHA-1 checksum file (`*.sha1sum`).

Archives are uploaded as release assets via `softprops/action-gh-release@v2`.

### Race Detection Workflow (`race.yaml`)

**Trigger:** On `push` to `main` and `pull_request` targeting `main`.

Runs on `ubuntu-latest` with Go 1.25.7.

```bash
go test -race ./domain/... ./app/... ./infrastructure/... -timeout=5m
```

This targets the core packages where concurrent operations are most likely:
- `domain/` — consensus engine, DAG topology, mining manager
- `app/` — application layer, RPC handlers, protocol messages
- `infrastructure/` — database, networking, configuration

---

## Build Optimization

### Stripped Binaries

Use `-ldflags="-s -w"` on all production builds:

| Flag | Effect |
|---|---|
| `-s` | Strip the symbol table |
| `-w` | Strip DWARF debug information |

This typically reduces binary size by 30–40%.

### Race Detector

The Go race detector (`-race`) instruments all memory accesses and detects data races at runtime. Use it during development and testing, **not** in production:

- Runtime overhead: 5–10× slower execution
- Memory overhead: 5–10× more memory
- Thread limit: ~8,192 goroutines on Linux (limited by `SIGRTMIN`)

Production builds should **never** include `-race`.

### Static Linking

For deployment on minimal container images or bare-metal servers, build fully static binaries. On Linux, combine:

```bash
CGO_ENABLED=0 go build -ldflags="-s -w -extldflags=-static" -tags netgo,osusergo
```

Docker builds use `musl-dev` in the build stage for CGO-dependent compilation, but the runtime image (`ubuntu:24.04`) only requires `ca-certificates`.

### Build Tags

| Tag | Purpose |
|---|---|
| `deadlock` | Instrument mutexes to detect deadlocks at runtime |
| `pebblegozstd` | Enable Zstandard compression in Pebble storage |
| `netgo` | Use pure-Go DNS resolver (required for static linking) |
| `osusergo` | Use pure-Go user/group lookup (required for static linking) |
| `ci` | CI-specific behavior (skip long-running tests) |

---

## Running Tests

### Full Test Suite

```bash
go test -timeout 20m ./...
```

### With Race Detection

```bash
go test -race -timeout 20m ./...
```

### With Coverage

```bash
go test -race -timeout 20m -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Integration Tests

Integration tests live in `testing/integration/`:

- `16_incoming_connections_test.go` — tests handling of 16 concurrent inbound connections
- `address_exchange_test.go` — tests peer address exchange protocol
- `basic_sync_test.go` — tests block synchronization between nodes

### Stability Tests

Long-running stability tests are in `stability-tests/`. Run them with:

```bash
cd stability-tests
./install_and_test.sh
```

Set `SKIP_LONG_STABILITY_TESTS=1` to run only the fast subset. Individual test suites:

| Suite | Description |
|---|---|
| `application-level-garbage` | Sends garbage application data to test robustness |
| `daa` | Difficulty adjustment algorithm validation |
| `htndsanity` | Comprehensive sanity checks on node behavior |
| `infra-level-garbage` | Sends garbage at the network protocol level |
| `many-tips` | Tests behavior under many competing tips |
| `mempool-limits` | Mempool capacity and eviction testing |
| `netsync` | Network synchronization under various DAG topologies |
| `orphans` | Orphan block handling |
| `reorg` | Chain reorganization scenarios |
| `rpc-idle-clients` | RPC stability with idle connections |
| `rpc-spam` | RPC resilience under spam |
| `rpc-stability` | Long-running RPC stability |
| `simple-sync` | Basic sync between two nodes |

### Test Data

Pre-configured test data for mainnet lives in `testdata/eyarod/eiyaro-mainnet/`.

---

## Code Quality Tools

### golangci-lint

Configuration in [`.golangci.yml`](../.golangci.yml):

```yaml
linters:
  default: none
  enable:
    - staticcheck
    - govet
    - errcheck
    - gosimple
    - ineffassign
    - revive
    - gosec
    - gocritic
    - misspell
  exclusions:
    paths:
      - "domain/consensus/utils/pow"

run:
  timeout: 5m
```

Note this configuration is for the `.golangci.yml` file — the actual linting in the Makefile uses `staticcheck` directly with a comprehensive list of check codes.

### gofmt

All code must pass `gofmt -s` (simplified formatting). The CI checks this with:

```bash
gofmt -l . | grep -v vendor
```

Run `make fmt` to auto-format all files before committing.

### go vet

```bash
go vet ./...
```

Run via `make vet`. The build scripts also embed `-vet` during race builds.

### go test -race

Running the race detector on the core packages (as the CI does):

```bash
go test -race ./domain/... ./app/... ./infrastructure/... -timeout=5m
```

---

## Usage Examples

### Full Build on Windows

```cmd
REM Clone the repository
git clone https://github.com/Hoosat-Oy/HTND.git
cd HTND

REM Download dependencies
go mod download

REM Production build
build.bat

REM Build with race detector for testing
build.bat race

REM Run basic tests
go test -timeout 20m ./...
```

### Full Build on Linux

```bash
# Clone the repository
git clone https://github.com/Hoosat-Oy/HTND.git
cd HTND

# Download dependencies
go mod download

# Build all binaries
./build.sh all

# Or use Makefile targets
make build
make build-race
make test-race
```

### Docker Build and Run

```bash
# Build the Docker image
docker build -t eiyaro:latest .

# Run the full node
docker run -d \
  --name eiyaro-node \
  -v /srv/eiyaro/data:/nonexistent/.eiyarod \
  -p 16110:16110 \
  -p 16111:16111 \
  eiyaro:latest

# View logs
docker logs -f eiyaro-node

# Run eiyaroctl inside the container
docker exec eiyaro-node /app/eiyaroctl getinfo

# Stop the node
docker stop eiyaro-node
```

### Running Full CI Pipeline Locally

```bash
# Run the exact CI sequence
make ci
```

Equivalent to:

```bash
make deps
make fmt-check
make vet
make lint
make test-race
make build-race
```

### Building for Release (Reproducing CI)

```bash
# Linux - static binary
GOEXPERIMENT=simd,jsonv2 \
  go build -v \
  -ldflags="-s -w -extldflags=-static" \
  -tags netgo,osusergo \
  -o bin/eyarod . ./cmd/...

# Windows
GOOS=windows GOARCH=amd64 GOEXPERIMENT=simd,jsonv2 \
  go build -v -ldflags="-s -w" -o bin/ ./cmd/...

# macOS
GOOS=darwin GOARCH=amd64 GOEXPERIMENT=simd,jsonv2 \
  go build -v -ldflags="-s -w" -o bin/ ./cmd/...
```

### Generating a Coverage Report

```bash
make test-coverage
# Open coverage.html in your browser
```

---

## FAQ

### Build fails with 'CGO is required' — what should I do?

Set `CGO_ENABLED=1` and ensure a C compiler is available (GCC via MinGW on Windows, gcc via build-essential on Linux, or Xcode Command Line Tools on macOS). Some dependencies like the `pebblegozstd` build tag require CGO. If you want a pure-Go build, set `CGO_ENABLED=0` and avoid CGO-dependent build tags, but note that Zstandard compression for PebbleDB will be unavailable.

### How do I cross-compile for a different platform?

Use Go's built-in cross-compilation by setting `GOOS` and `GOARCH` environment variables. For example, to build a Windows binary from Linux: `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/eyarod.exe ./cmd/eyarod`. The release pipeline already produces binaries for Linux (amd64, static), Windows (amd64), and macOS (amd64).

### Why is my binary so large?

The unstripped debug binary can be 100+ MB. Production builds use `-ldflags="-s -w"` to strip the symbol table and DWARF debug information, reducing size by 30–40%. Docker builds additionally enable `simd` and `jsonv2` Go experiments which add some size. For minimal deployment, always use stripped production builds.

### Can I build without Docker?

Yes. Docker is only needed if you want a containerized deployment. Use `build.bat` (Windows) or `build.sh` (Linux/macOS) for native builds, or simply run `go build -ldflags="-s -w" -o bin/eyarod ./cmd/eyarod`. The Makefile provides `make build`, `make build-race`, and `make production` targets. The CI pipeline tests both native and Docker builds.

### Which build tags should I use for production?

The Docker production build uses `-tags "deadlock pebblegozstd"`. The `deadlock` tag instruments mutexes for deadlock detection (useful in production for catching concurrency bugs), and `pebblegozstd` enables Zstandard compression in PebbleDB for better storage efficiency. For static Linux builds, also use `-tags netgo,osusergo` to avoid CGO DNS and user lookups.