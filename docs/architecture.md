# Eiyaro System Architecture

## Overview

Eiyaro is a modular, GhostDAG-based blockchain node implementation written in Go. It is organized into five distinct layers with strict dependency direction: `cmd/` → `app/` → `domain/` → `infrastructure/` → `util/`. Each layer encapsulates a specific responsibility, enabling independent testing, maintenance, and evolution of components.

The architecture follows Clean Architecture principles: domain logic has no awareness of networking or database details, infrastructure provides pluggable backends (LevelDB / PebbleDB), and the application layer orchestrates components through a central `ComponentManager`.

---

## Layer Architecture

```
┌──────────────────────────────────────────────────────────┐
│                   cmd/ (Entry Points)                     │
│  eyarod | eiyaroctl | eiyarowallet | miner | genkeypair  │
├──────────────────────────────────────────────────────────┤
│                  app/ (Application Layer)                 │
│  Component Manager | RPC | Protocol | AppMessage         │
├──────────────────────────────────────────────────────────┤
│                domain/ (Business Logic)                   │
│  Consensus (GhostDAG) | Mining Manager | UTXO Index      │
│  DAG Config | Prefix Manager | Consensus Reference       │
├──────────────────────────────────────────────────────────┤
│           infrastructure/ (Infrastructure)                │
│  DB (LevelDB/PebbleDB) | Network (P2P/NAT/Seeds)        │
│  Config | Logger | OS (Signals/Windows Service)          │
├──────────────────────────────────────────────────────────┤
│                 util/ (Utilities)                         │
│  Bech32 | txscript | Blake2b | Difficulty | LRU Cache   │
│  Staging | Math | Random | Network Parsing               │
└──────────────────────────────────────────────────────────┘
```

Dependency direction: upper layers depend on lower layers. Utilities have no internal dependencies. Infrastructure depends only on utilities. Domain depends on utilities and infrastructure (primarily the `database.Database` interface). The application layer composes domain and infrastructure into runnable services.

---

## Layer Details

### `cmd/` — Entry Points

Each subdirectory is a standalone `main` package producing a separate binary.

| Binary | Package | Purpose |
|--------|---------|---------|
| `eyarod` | `cmd/eyarod` | Full node daemon. Starts the component manager, opens the database, and blocks on OS signals. Supports Windows service mode. See [eyarod.md](eyarod.md). |
| `eiyaroctl` | `cmd/eiyaroctl` | CLI RPC client. Uses reflection-based command dispatch over gRPC/JSON. Connects to an eyarod node. See [eiyaroctl.md](eiyaroctl.md). |
| `eiyarowallet` | `cmd/eiyarowallet` | HD wallet implementing BIP39/BIP32 with Schnorr and ECDSA signatures. Supports daemon mode with gRPC API. See [wallet.md](wallet.md). |
| `miner` | `cmd/miner` | CPU miner implementing the NogoPow algorithm. Fetches block templates via RPC and submits solved blocks. See [miner.md](miner.md). |
| `genkeypair` | `cmd/genkeypair` | Standalone key pair generator for offline address creation. |

The `eyarod` entry point ([main.go](file://d:/Eiyaro/Eiyaro/cmd/eyarod/main.go)) sets runtime environment variables (`GOGC`, `GOMEMLIMIT`), optionally starts a pprof HTTP server, then delegates to `app.Start()`. On Windows, service mode is detected and handled via `winservice.WinServiceMain`.

### `app/` — Application Layer

The application layer wires together all subsystems and manages their lifecycle.

**Component Manager** ([component_manager.go](file://d:/Eiyaro/Eiyaro/app/component_manager.go)) is the central orchestrator. It:
1. Creates the `domain.Domain` instance (consensus + mining manager)
2. Creates the network adapter (`netadapter.NetAdapter`) for P2P and RPC transport
3. Creates the address manager for peer address storage
4. Optionally creates the UTXO index
5. Creates the connection manager for P2P peer connections
6. Creates the protocol manager for P2P protocol flows
7. Creates and wires the RPC manager

Lifecycle: `Start()` → net adapter starts, connection manager begins peering. `Stop()` → RPC stats stop, connection manager stops, net adapter stops, protocol manager closes, consensus event channel closes.

**RPC Manager** ([app/rpc/manager.go](file://d:/Eiyaro/Eiyaro/app/rpc/manager.go)):
- Handles JSON-RPC over gRPC with protobuf `EiyaroMessage` envelopes
- Listens on the consensus events channel for `BlockAdded` and `VirtualChangeSet` events
- Converts domain events to RPC notifications (block added, virtual chain change, UTXO change)
- Delegates to `NotificationManager` for push-style subscriber notifications

See [rpc-api.md](rpc-api.md) for the full API reference.

**Protocol Manager** ([app/protocol/manager.go](file://d:/Eiyaro/Eiyaro/app/protocol/manager.go)):
- Manages P2P protocol flows per peer connection
- Implements version negotiation (protocol version 7)
- Handles handshake (`Version` → `VerAck` → `Ready`)
- Manages IBD (Initial Block Download) peer selection
- Provides callbacks for new block templates and pruning point UTXO set overrides

**AppMessage** (`app/appmessage/`):
- Defines protobuf message types for all RPC and P2P commands
- Provides domain-to-RPC conversion functions (`DomainBlockToRPCBlock`, etc.)
- Implements message serialization for the wire protocol

### `domain/` — Business Logic Layer

The domain layer contains all blockchain-specific logic with no network or RPC awareness.

**Domain Interface** ([domain/domain.go](file://d:/Eiyaro/Eiyaro/domain/domain.go)):
```go
type Domain interface {
    MiningManager() miningmanager.MiningManager
    Consensus() externalapi.Consensus
    StagingConsensus() externalapi.Consensus
    InitStagingConsensusWithoutGenesis() error
    CommitStagingConsensus() error
    DeleteStagingConsensus() error
    ConsensusEventsChannel() chan externalapi.ConsensusEvent
}
```

The staging consensus pattern supports non-disruptive database migrations: a second consensus instance is initialized with an inactive database prefix, data is migrated, and the prefix is atomically swapped via `CommitStagingConsensus()`.

**Consensus Engine** ([domain/consensus/](file://d:/Eiyaro/Eiyaro/domain/consensus/)):
- GhostDAG protocol implementation — orders blocks in a DAG into blue (confirmed) and red (orphaned) sets
- Block validation pipeline: header isolation → body isolation → header context → body context
- Block insertion with atomic staging area pattern
- UTXO diff algebra — stores per-block UTXO changes instead of full snapshots
- Multiset UTXO commitment for light client verification
- Difficulty Adjustment Algorithm (DAA) with window-based retargeting
- Finality determination (24-hour finality window)
- Pruning with pruning proofs for header-first IBD
- Sub-linear reachability queries via reachability tree intervals

See [consensus.md](consensus.md) for the complete consensus reference.

**DAG Config** ([domain/dagconfig/](file://d:/Eiyaro/Eiyaro/domain/dagconfig/)):
- Six network configurations: Mainnet, Testnet, Testnet-B5, Testnet-B10, Devnet, Simnet
- Network parameters: K=18, 20s block time, 2641-block DAA window, 500K block mass
- Genesis block definitions with per-network coinbase payloads
- Economic model: 8 EY base subsidy, deflationary phase after DAA score 360, 0.1 EY floor
- Address prefixes: `eiyaro:` (mainnet), `eiyarotest:` (testnet), `eiyarosim:` (simnet)

See [network-params.md](network-params.md) for the complete parameters reference.

**Mining Manager** ([domain/miningmanager/](file://d:/Eiyaro/Eiyaro/domain/miningmanager/)):
- Block template construction from mempool transactions
- Mempool management with orphan transaction support, RBF (Replace-By-Fee), compound transaction rate limiting, and wallet freezing
- Fee estimation via priority buckets

**UTXO Index** ([domain/utxoindex/](file://d:/Eiyaro/Eiyaro/domain/utxoindex/)):
- Optional address-to-UTXO mapping for fast balance queries
- Updated incrementally from `VirtualChangeSet` consensus events
- Required for `getUtxosByAddresses` and `getBalanceByAddress` RPC methods

**Prefix Manager** ([domain/prefixmanager/](file://d:/Eiyaro/Eiyaro/domain/prefixmanager/)):
- Manages active/inactive database key prefixes for the staging consensus pattern
- Supports atomic prefix swapping without database downtime

### `infrastructure/` — Infrastructure Layer

The infrastructure layer provides platform-level services with well-defined interfaces.

**Database** (`infrastructure/db/`):
- Common `database.Database` interface with `Put`, `Get`, `Delete`, `Cursor`, `Begin` operations
- Two backends: **LevelDB** (`ldb` package, 256 MiB cache) and **PebbleDB** (`pebble` package, 2 GiB cache)
- PebbleDB is the default; configurable via `HTND_PEBBLE_CACHE_MB`, `HTND_MEMTABLE_SIZE_MB`, and other environment variables
- Key-value store with prefix-based namespacing for multi-consensus support

**Network** (`infrastructure/network/`):
- `netadapter` — Network adapter wrapping gRPC transport for both P2P and RPC
- `connmanager` — Connection lifecycle, peering targets (8 outbound, 500 max inbound), ban management
- `addressmanager` — Peer address persistence and DNS seeding
- `netadapter/router` — Route-based message dispatch with per-route FIFO queues
- `netadapter/id` — Cryptographic node identity generation

**Config** ([infrastructure/config/](file://d:/Eiyaro/Eiyaro/infrastructure/config/)):
- INI-format configuration with CLI flag overlay
- Network auto-detection from flags (`--testnet`, `--devnet`, `--simnet`)
- Environment variable expansion (`$VARIABLE`, `~`)
- Auto-creation of default config file on first run

See [configuration.md](configuration.md) for the complete configuration reference.

**Logger** (`infrastructure/logger/`):
- Structured logging with subsystem tags
- Per-subsystem log level control (`--loglevel=PROT=debug,NET=info`)
- File rotation and dual output (stdout + log files)

**OS** (`infrastructure/os/`):
- `signal` — SIGINT/SIGTERM handling with graceful shutdown
- `execenv` — File descriptor limits and runtime initialization
- `winservice` — Windows Service Control Manager integration

### `util/` — Utilities

Zero-dependency utilities used throughout the codebase.

| Package | Purpose |
|---------|---------|
| `bech32` | BIP173 Bech32 address encoding/decoding with network-specific HRPs |
| `txscript` | Transaction script VM (OP codes, signature verification, multisig) |
| `blake2b` | Blake2b hashing with domain separation tags (Merkle tree, transaction signing) |
| `difficulty` | Compact target ↔ big integer conversion |
| `staging` | Atomic staging area pattern for batched database commits |
| `lru` | Generic LRU cache with configurable capacity |
| `math` | Overflow-safe integer arithmetic (`SafeAdd`, `SafeMul`) |
| `random` | Cryptographically secure random generation via `crypto/rand` |
| `mstime` | Millisecond-precision monotonic time utilities |
| `txmass` | Transaction mass (weight) computation from script and signature operations |
| `panics` | Panic recovery with structured logging |
| `profiling` | Heap profiling and pprof HTTP server helpers |
| `network` | Network address parsing and listener configuration |

---

## Data Flow

### Block Lifecycle

```
Miner                          eyarod (Full Node)

GetBlockTemplate ─────────────► domain.MiningManager.GetBlockTemplate()
                                │
                    ┌───────────┘
                    ▼
              BuildBlockTemplate()
              ├─ Select DAG tips as parents
              ├─ Build coinbase tx
              ├─ Select mempool txs
              └─ Return block template + PoW state
                                │
    ◄───────────────────────────┘
    │
Mine (NogoPow)
├─ Increment nonce
├─ Blake3(header) → seed
├─ Matrix A × Matrix B (256×256 fixed-point)
├─ Blake3(result, seed) → powHash
└─ if powHash < target: found!
    │
SubmitBlock ──────────────────► domain.Consensus.ValidateAndInsertBlock()
                                │
                    ┌───────────┘
                    ▼
              Validation Pipeline
              ├─ Validate header in isolation
              ├─ Resolve parents via BlockParentBuilder
              ├─ Run GhostDAG → blue score, blue work, merge sets
              ├─ Validate header in context (PoW, difficulty, parents)
              ├─ Validate body in isolation (mass, txs)
              ├─ Validate body in context (UTXO ops, coinbase)
              ├─ Update virtual state
              ├─ Compute finality point
              └─ Commit via staging area
                                │
                    ┌───────────┘
                    ▼
              Consensus Events Channel
              ├─ BlockAdded event → RPC notifications
              └─ VirtualChangeSet event → UTXO index update
```

### Transaction Lifecycle

```
Wallet                          eyarod

CreateUnsignedTx
├─ Select UTXOs
├─ Build inputs/outputs
└─ Leave signature slots empty
    │
Sign (offline/daemon)
├─ Derive private keys (BIP32)
├─ Compute SigHashAll
└─ Schnorr/ECDSA sign each input
    │
SubmitTransaction ────────────► app/protocol.Manager.AddTransaction()
                                │
                    ┌───────────┘
                    ▼
              Mempool
              ├─ Validate transaction
              ├─ Check for doublespends
              ├─ Add to mempool (or orphan pool)
              └─ Relay to P2P peers
                                │
               (wait for mining)
                                │
                    ┌───────────┘
                    ▼
              Included in Block Template
              → Block mined
              → Block inserted into DAG
              → UTXO set updated
              → Coinbase matures after 100 blocks
```

---

## Key Design Patterns

### Staging Area Pattern
All consensus database mutations use a staging area ([util/staging/](file://d:/Eiyaro/Eiyaro/util/staging/)) for atomicity. Changes are staged across multiple stores, then committed atomically via `CommitAllChanges`. If any validation step fails, the staging area is discarded — no partial state is ever persisted. This is critical for multi-step block validation where partial updates would corrupt the DAG.

### Factory Pattern
The consensus engine is constructed via `consensus.NewFactory()` ([domain/consensus/factory.go](file://d:/Eiyaro/Eiyaro/domain/consensus/factory.go)). The factory creates all internal process managers (GhostDAG, DAG topology, difficulty, pruning, etc.) and wires them together. This allows dependency injection and testability.

### Interface-Based Modularity
Key abstractions are defined as Go interfaces:
- `externalapi.Consensus` — 60+ method consensus API ([consensus.go](file://d:/Eiyaro/Eiyaro/domain/consensus/model/externalapi/consensus.go))
- `domain.Domain` — top-level domain facade
- `database.Database` — database backend abstraction (LevelDB / PebbleDB)
- `miningmanager.MiningManager` — mining and mempool operations

This enables backend swapping (e.g., LevelDB ↔ PebbleDB) and mock-based testing.

### Event-Driven Notifications
Consensus emits events on a buffered channel (`BlockAdded`, `VirtualChangeSet`). The RPC manager consumes these events and converts them to subscriber notifications (block added, virtual chain changed, UTXOs changed). This decouples consensus internals from the notification system.

### Mutex-Protected Shared State
The consensus engine uses a single global mutex for all state-modifying operations. Block insertion is single-threaded, with virtual resolution releasing the lock in 100-block chunks to prevent starvation. The `ComponentManager` uses `atomic.Int32` for start/shutdown guards.

### Prefix-Based Database Namespacing
Multiple consensus instances can share a single database by using different key prefixes. The `PrefixManager` manages active/inactive prefixes, enabling the staging consensus pattern for non-disruptive migrations.

---

## Cross-Reference

This document provides a high-level architectural overview. For detailed component documentation, refer to:

| Document | Content |
|----------|---------|
| [eyarod.md](eyarod.md) | Full node daemon — startup, flags, data directory, Windows service, signal handling |
| [eiyaroctl.md](eiyaroctl.md) | CLI RPC client — all commands, parameters, usage examples |
| [wallet.md](wallet.md) | HD wallet — BIP39/BIP32, Schnorr/ECDSA, multisig, daemon mode, gRPC API |
| [miner.md](miner.md) | CPU miner — NogoPow algorithm, mining loop, template management, hashrate reporting |
| [consensus.md](consensus.md) | GhostDAG consensus — algorithm, block validation, UTXO model, finality, pruning, reachability |
| [network-params.md](network-params.md) | Network parameters — all 6 networks, genesis blocks, subsidy schedule, address prefixes |
| [configuration.md](configuration.md) | Configuration — all CLI flags, config file format, environment variables, precedence rules |
| [rpc-api.md](rpc-api.md) | JSON-RPC API — all methods, message format, notification subscriptions, curl examples |
| [nogopow.md](nogopow.md) | NogoPow proof-of-work — 256×256 matrix algebra, Blake3, Xoshiro256++, PI controller |

### Key Source Files

| File | Role |
|------|------|
| [main.go](file://d:/Eiyaro/Eiyaro/cmd/eyarod/main.go) | eyarod entry point — runtime config, profiler |
| [app.go](file://d:/Eiyaro/Eiyaro/app/app.go) | Application bootstrap — DB open, component manager |
| [component_manager.go](file://d:/Eiyaro/Eiyaro/app/component_manager.go) | Central orchestrator — lifecycle, subsystem wiring |
| [domain.go](file://d:/Eiyaro/Eiyaro/domain/domain.go) | Domain facade — consensus, mining, staging |
| [consensus.go](file://d:/Eiyaro/Eiyaro/domain/consensus/model/externalapi/consensus.go) | Consensus interface — 60+ methods |
| [factory.go](file://d:/Eiyaro/Eiyaro/domain/consensus/factory.go) | Consensus factory — construction, config |
| [manager.go (protocol)](file://d:/Eiyaro/Eiyaro/app/protocol/manager.go) | P2P protocol — flows, handshake, IBD |
| [manager.go (rpc)](file://d:/Eiyaro/Eiyaro/app/rpc/manager.go) | RPC manager — events, notifications |
| [protocol.go](file://d:/Eiyaro/Eiyaro/app/protocol/protocol.go) | Protocol router — flow registration, version negotiation |
| [config.go](file://d:/Eiyaro/Eiyaro/infrastructure/config/config.go) | Configuration struct and loader |
| [params.go](file://d:/Eiyaro/Eiyaro/domain/dagconfig/params.go) | Network parameter definitions |
| [bech32.go](file://d:/Eiyaro/Eiyaro/util/bech32/bech32.go) | Bech32 address encoding |
| [pow.go](file://d:/Eiyaro/Eiyaro/domain/consensus/utils/pow/pow.go) | NogoPow state, matrix generation, verification |

---

## FAQ

### Which layer should I modify to add a new feature?

It depends on the feature's scope. **Consensus-level features** (new validation rules, transaction types, DAG algorithms) belong in `domain/consensus/`. **P2P protocol features** (new message types, flow handlers) go in `app/protocol/`. **RPC methods** are added in `app/rpc/`. **New CLI tools** become new packages under `cmd/`. Infrastructure backends (storage, networking) live in `infrastructure/`. The dependency rule is strict: higher layers can depend on lower layers, never the reverse.

### How do components communicate with each other?

The primary communication channel is the **consensus events channel** (`chan externalapi.ConsensusEvent`). When a block is validated and inserted, consensus emits `BlockAdded` and `VirtualChangeSet` events. The RPC manager consumes these events and converts them to subscriber notifications. For P2P protocol flows, components communicate through the `Router` abstraction, which maps message types to named route channels.

### What is the difference between cmd/ and app/?

`cmd/` contains standalone `main` packages that produce separate binaries (eyarod, eiyaroctl, eiyarowallet). These are thin entry points that initialize runtime settings and delegate to the application layer. `app/` contains the composable application logic — the `ComponentManager` orchestrator, protocol flows, RPC handlers, and message definitions. The `app/` layer is importable; `cmd/` packages produce executables.

### How do I add a new RPC method?

Define the protobuf request and response messages in `app/appmessage/`, register a new handler function in `app/rpc/manager.go` or a dedicated handler file, and wire it into the gRPC service registration in the net adapter. Existing handlers follow a consistent pattern: parse request → delegate to domain → format response. See existing methods like `GetBalance` or `GetBlock` for reference implementations.

### What is the staging consensus pattern used for?

The staging consensus manages two consensus instances with different database key prefixes. This enables non-disruptive database migrations: a new consensus instance is initialized with an inactive prefix, data is migrated to it, and then the prefix is atomically swapped via `CommitStagingConsensus()`. The pattern guarantees that migration failures leave the original data intact, since `DeleteStagingConsensus()` discards all staged changes.