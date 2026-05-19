# Eiyaro Full Node (eyarod)

## Overview

**eyarod** is the full node daemon for the Eiyaro blockchain network. It implements the GhostDAG consensus protocol, a scalable generalization of Nakamoto consensus that supports high block rates by allowing blocks to reference multiple parents and ordering them via a topological algorithm.

The node connects to the peer-to-peer network, validates and relays blocks and transactions, maintains a local copy of the block DAG, and exposes an RPC interface for applications and wallets to interact with the network.

eyarod is written in Go and can run on Linux, macOS, and Windows systems. It supports both LevelDB and PebbleDB as database backends for storing the block DAG and consensus state.

## Quick Start

### Starting a Mainnet Node

```bash
eyarod
```

This launches the node with all default settings: mainnet, data stored under `~/.eyarod/eiyaro-mainnet/`, RPC listening on `0.0.0.0:42420`, and P2P listening on `0.0.0.0:42421`.

### Starting a Testnet Node

```bash
eyarod --testnet
```

This connects to the Eiyaro testnet, storing data under `~/.eyarod/eiyaro-testnet/`, with RPC on `0.0.0.0:42422` and P2P on `0.0.0.0:42423`.

### Expected Output on Successful Startup

```
[INF] EYARO Version 1.0.0-<commit-hash>
[INF] Loading pebble database from '/home/user/.eyaro/eiyaro-mainnet/datadir2'
[INF] UTXO index started
...
```

If the node is syncing blocks during Initial Block Download (IBD), you will see periodic log messages indicating sync progress. Once fully synced, the node will quietly process new blocks and transactions.

## Command-Line Flags

All flags can be specified on the command line or in the [configuration file](#configuration-file). Command-line values take precedence over configuration file values.

### Core Settings

| Flag | Type | Default | Description |
|---|---|---|---|
| `--version` (`-V`) | bool | — | Display version information and exit |
| `--configfile` (`-C`) | string | `~/.eyaro/eiyarod.conf` | Path to configuration file |
| `--appdir` (`-b`) | string | `~/.eyaro` | Directory to store all application data |
| `--logdir` | string | `<appdir>/<network>/logs` | Directory to write log output |
| `--loglevel` (`-d`) | string | `info` | Logging level: `trace`, `debug`, `info`, `warn`, `error`, `critical`. Supports per-subsystem levels via `subsystem=level,...`. Use `show` to list available subsystems |
| `--dbtype` | string | `pebble` | Database backend: `leveldb` or `pebble` |
| `--profile` | string | — | Enable HTTP pprof server on given port (must be 1024–65535) |
| `--reset-db` | bool | false | Delete all blockchain data before starting |
| `--archival` | bool | false | Run as an archival node — do not prune old block data (heavy disk usage) |
| `--data-retention-hours` | uint64 | 0 | Minimum hours of chain data to keep before pruning. 0 uses standard pruning depth rules |
| `--pruning-interval-hours` | uint64 | 0 | Minimum hours between pruning point updates. 0 prunes at every pruning point update |
| `--utxoindex` | bool | false | Enable the UTXO index for address-based queries |

### Network Selection

Only one network flag may be specified at a time. The default is mainnet.

| Flag | Network | RPC Port | P2P Port | Data Directory |
|---|---|---|---|---|
| _(none)_ | `eiyaro-mainnet` | `42420` | `42421` | `~/.eyaro/eiyaro-mainnet/` |
| `--testnet` | `eiyaro-testnet` | `42422` | `42423` | `~/.eyaro/eiyaro-testnet/` |
| `--testnet-b5` | `eiyaro-testnet-b5` | `42422` | `42423` | `~/.eyaro/eiyaro-testnet-b5/` |
| `--testnet-b10` | `eiyaro-testnet-b10` | `42422` | `42423` | `~/.eyaro/eiyaro-testnet-b10/` |
| `--simnet` | `eiyaro-simnet` | `42424` | `42425` | `~/.eyaro/eiyaro-simnet/` |
| `--devnet` | `eiyaro-devnet` | `42426` | `42427` | `~/.eyaro/eiyaro-devnet/` |

### P2P Networking

| Flag | Type | Default | Description |
|---|---|---|---|
| `--listen` | []string | `0.0.0.0:<network-port>` | Interfaces/ports to listen on for incoming P2P connections |
| `--nolisten` | bool | false | Disable listening for incoming P2P connections (automatic with `--connect` or `--proxy` without `--listen`) |
| `--addpeer` (`-a`) | []string | — | Add a peer to connect with at startup (cannot mix with `--connect`) |
| `--connect` | []string | — | Connect exclusively to specified peers (disables DNS seeding and outbound peering) |
| `--externalip` | []string | — | IP addresses to advertise to peers as our listening addresses |
| `--outpeers` | int | 8 | Target number of outbound peer connections |
| `--maxinpeers` | int | 500 | Maximum number of inbound peer connections |
| `--nodnsseed` | bool | false | Disable DNS-based peer discovery |
| `--dnsseed` | string | — | Override default DNS seeds with a custom hostname |
| `--grpcseed` | string | — | Hostname of a gRPC server for peer seeding |
| `--proxy` | string | — | Connect via SOCKS5 proxy (e.g., `127.0.0.1:9050`) |
| `--proxyuser` | string | — | Username for the SOCKS5 proxy |
| `--proxypass` | string | — | Password for the SOCKS5 proxy |
| `--disallow-loopback-p2p` | bool | false | Prevent outbound P2P connections to loopback addresses |
| `--upnp` | bool | false | Use UPnP to map the listening port through NAT |

### Peer Management

| Flag | Type | Default | Description |
|---|---|---|---|
| `--enablebanning` | bool | false | Enable automatic banning of misbehaving peers |
| `--banduration` | time.Duration | `24h` | Duration to ban misbehaving peers (minimum 1s) |
| `--banthreshold` | uint32 | 100 | Ban score threshold before disconnecting and banning |
| `--whitelist` | []string | — | IP addresses or CIDR networks exempt from banning |

### RPC Server

| Flag | Type | Default | Description |
|---|---|---|---|
| `--rpclisten` | []string | `0.0.0.0:<network-rpc-port>` | Interfaces/ports for RPC connections |
| `--norpc` | bool | false | Disable the built-in RPC server entirely |
| `--saferpc` | bool | false | Disable RPC commands that mutate node state |
| `--rpccert` | string | `~/.eyaro/rpc.cert` | Path to the RPC TLS certificate file |
| `--rpckey` | string | `~/.eyaro/rpc.key` | Path to the RPC TLS key file |
| `--rpcmaxclients` | int | 500 | Maximum concurrent RPC clients |
| `--rpcmaxwebsockets` | int | 250 | Maximum concurrent WebSocket connections |
| `--rpcmaxconcurrentreqs` | int | 10000 | Maximum concurrent RPC requests being processed |

### Transaction & Mempool

| Flag | Type | Default | Description |
|---|---|---|---|
| `--minrelaytxfee` | float64 | `1e-5` | Minimum transaction fee (HTN/kB) for relay consideration |
| `--maxorphantx` | uint64 | 100 | Maximum orphan transactions held in memory |
| `--blockmaxmass` | uint64 | `10_000_000` | Maximum block mass used when building template blocks (range: 1000–10000000) |
| `--blocksonly` | bool | false | Do not accept or relay transactions from remote peers |
| `--relaynonstd` | bool | false | Relay non-standard transactions |
| `--rejectnonstd` | bool | false | Reject non-standard transactions (cannot combine with `--relaynonstd`) |

### Compound Transaction Rate Limiting

| Flag | Type | Default | Description |
|---|---|---|---|
| `--max-compound-tx-per-minute` | uint64 | 10 | Maximum compound transactions per address per minute |
| `--compound-tx-ratelimit-window` | uint64 | 1 | Rate limit sliding window in minutes |
| `--compound-tx-inputs-threshold` | uint64 | 21 | Minimum number of inputs to classify a transaction as compound |

### Wallet Freezing

| Flag | Type | Default | Description |
|---|---|---|---|
| `--freeze-address` | []string | — | Address to freeze (may be specified multiple times) |

### IBD (Initial Block Download)

| Flag | Type | Default | Description |
|---|---|---|---|
| `--disable-ibd-timeout` | bool | false | Disable IBD timeout (node won't disconnect slow IBD peers) |
| `--ibd-timeout` | time.Duration | `120m` | Maximum time allowed for IBD before disconnecting and retrying another peer |
| `--nearly-synced-ibd-timeout` | time.Duration | `10m` | IBD timeout when the node is near the chain tip |
| `--ibd-dequeue-timeout` | time.Duration | `1m` | Maximum time to wait for a block to dequeue during IBD |
| `--min-headers-per-second` | int | 10 | Minimum headers per second required from an IBD peer |
| `--min-blocks-per-second` | int | 10 | Minimum blocks per second required from an IBD peer |

### Performance & Tuning

| Flag | Type | Default | Description |
|---|---|---|---|
| `--sigcachemaxsize` | uint | 100000 | Maximum entries in the signature verification cache |
| `--maxutxocachesize` | uint64 | `5_000_000_000` | Maximum UTXO cache size in bytes (loaded from disk into RAM) |
| `--utxo-default-max-limit` | uint32 | 0 | Default maximum limit for UTXO pull requests |
| `--use-hoohash-c-library` | bool | `true` (Linux ARM64) | Use the C implementation of hoohash for PoW validation (block version ≥ 5) |

### P2P Protocol

| Flag | Type | Default | Description |
|---|---|---|---|
| `--uacomment` | []string | — | Custom comment appended to the user agent string (per BIP 14) |
| `--force-same-version` | bool | false | Disconnect peers whose protocol version differs from ours |
| `--nopeerbloomfilters` | bool | false | Disable Bloom filter support for light clients |
| `--protocol-version` | uint32 | 7 | Override the default P2P protocol version |

### Developer & Testing (Hidden)

| Flag | Type | Default | Description |
|---|---|---|---|
| `--deletion-depth` | uint64 | 0 | Custom pruning deletion depth (multiplies standard pruning depth) |
| `--allow-submit-block-when-not-synced` | bool | false | Accept blocks from RPC while the node is not synced |
| `--enable-sanity-check-pruning-utxo` | bool | false | Verify UTXO set matches the UTXO commitment when moving the pruning point |
| `--override-dag-params-file` | string | — | Override DAG parameters via a JSON file (devnet only) |

### Windows Service

| Flag | Type | Default | Description |
|---|---|---|---|
| `--service` (`-s`) | string | — | Service command: `install`, `remove`, `start`, or `stop` |

## Configuration File

eyarod looks for a configuration file at `~/.eyaro/eiyarod.conf` by default. Use `--configfile` to specify an alternative path.

If the configuration file does not exist on first run, eyarod automatically creates one with sensible defaults at the default path. The file uses INI format:

```ini
[Application Options]
loglevel=info
rpclisten=0.0.0.0:42420
listen=0.0.0.0:42421
```

## Data Directory Structure

The data directory layout depends on the active network. For a mainnet node with default settings:

```
~/.eyaro/
├── eiyarod.conf                  # Configuration file
├── rpc.cert                      # RPC TLS certificate
├── rpc.key                       # RPC TLS private key
└── eiyaro-mainnet/               # Network-specific data (namespaced by network)
    ├── datadir2/                 # Consensus database (LevelDB or PebbleDB)
    └── logs/                     # Log files
        ├── eiyarod.log           # Main log
        └── eiyarod_err.log       # Error log
```

For testnet (`--testnet`), the network subdirectory is `eiyaro-testnet/`. The pattern `~/.eyaro/<network-name>/` is used for all networks.

### Database Backend

The `datadir2/` directory stores the complete consensus database using either LevelDB or PebbleDB (default). The database contains all block headers, block data, transaction data, UTXO set information, and consensus state.

#### Cache Sizes

| Backend | Default Cache Size |
|---|---|
| LevelDB | 256 MiB |
| PebbleDB | 2048 MiB (2 GiB) |

## Signal Handling

eyarod performs a graceful shutdown when it receives an operating system interrupt signal:

| Signal | Trigger | Behavior |
|---|---|---|
| `SIGINT` | `Ctrl+C` in terminal | Initiates graceful shutdown |
| `SIGTERM` | `kill` command / system shutdown | Initiates graceful shutdown |

On receiving the first termination signal, eyarod begins shutting down all subsystems in order:

1. Stop RPC statistics tracking
2. Stop the connection manager (disconnect all peers)
3. Stop the network adapter
4. Close the protocol manager
5. Close the consensus event channel
6. Close the database

A **2-minute shutdown timeout** is enforced. If graceful shutdown exceeds this limit, the node terminates forcefully with a critical log message.

If a second interrupt signal is received during shutdown, eyarod logs a message acknowledging the repeated signal and continues shutting down — it does not hang.

The `ShutdownRequestChannel` is also used internally: the RPC server can request shutdown programmatically (e.g., via the `stop` RPC command or the Windows Service Control Manager).

## Windows Service Mode

On Windows, eyarod can run as a system service managed by the Windows Service Control Manager (SCM).

### Service Name

- **Name**: `eyarodsvc`
- **Display Name**: `eyarod Service`
- **Description**: Downloads and stays synchronized with the Eiyaro blockDAG and provides DAG services to applications.

### Service Commands

```bash
# Install the service
eyarod --service=install

# Start the service
eyarod --service=start

# Stop the service
eyarod --service=stop

# Remove the service
eyarod --service=remove
```

### Service Lifecycle

When running as a service:

1. The service reports `StartPending` to the SCM while initializing.
2. The main node logic (`eyarodApp.main`) runs in a separate goroutine to allow the service to start quickly.
3. Once started, the service reports `Running` and accepts `Stop` and `Shutdown` control commands.
4. When a stop or shutdown is requested through the SCM, the service sends a signal to `ShutdownRequestChannel`, triggering the standard graceful shutdown sequence.
5. Service startup information (version, config file path, data directory, logs directory) is logged to the Windows Event Log.
6. On service exit, the SCM is notified that the service has `Stopped`.

### Automatic Detection

When `eyarod` is started directly (not via the SCM), the application detects that it is running on Windows and calls `WinServiceMain`. If the process was started by the SCM, `WinServiceMain` runs the service loop and returns `true` (isService), causing `Start()` to exit immediately. Otherwise, execution falls through to standard interactive operation.

## Profiling

eyarod includes built-in Go pprof profiling support. Enable it with the `--profile` flag:

```bash
eyarod --profile=6060
```

This starts an HTTP server on the specified port with the standard Go pprof endpoints:

- `/debug/pprof/` — Index page listing available profiles
- `/debug/pprof/profile` — CPU profile
- `/debug/pprof/heap` — Heap (memory) profile
- `/debug/pprof/goroutine` — Goroutine stack traces
- `/debug/pprof/trace` — Execution trace

Additionally, eyarod automatically tracks heap usage by writing periodic heap profiles into `<appdir>/heap-<timestamp>.prof` for offline analysis.

For programmatic profiling, set the `EYAROD_PROFILER` environment variable (to any non-empty value) before starting. This also enables block and mutex profiling, and starts the pprof server on the port specified by `PROFILER_PORT` (default `8080`):

```bash
EYAROD_PROFILER=1 PROFILER_PORT=6060 eyarod
```

### Memory Tuning

Two environment variables control Go's garbage collector behavior:

| Variable | Default | Description |
|---|---|---|
| `GOGC` | `100` | GC target percentage (100 means heap doubles before GC triggers) |
| `GOMEMLIMIT` | `8000000000` (8 GB) | Soft memory limit in bytes for the Go runtime |

## Usage Examples

### Start a Mainnet Node with UTXO Index

The UTXO index enables address-based queries via the RPC API (e.g., `getUtxosByAddresses`):

```bash
eyarod --utxoindex
```

### Connect to Specific Peers

To connect exclusively to known peers (disables DNS seed discovery and outbound peering to random nodes):

```bash
eyarod --connect=192.168.1.100:42421 --connect=10.0.0.50:42421
```

### Add Persistent Peers (Alongside Normal Peering)

To add specific peers while still participating in normal peer discovery:

```bash
eyarod --addpeer=trusted-node.example.com:42421
```

### Run on Testnet with Increased Logging

```bash
eyarod --testnet --loglevel=debug
```

### Run an Archival Node

Archival nodes retain all historical block data (never prune). This consumes significant disk space:

```bash
eyarod --archival
```

### Reset Database and Start Fresh

Delete all blockchain data and re-sync from genesis:

```bash
eyarod --reset-db
```

### Custom Data and Log Directories

```bash
eyarod --appdir=/data/eyaro --logdir=/var/log/eyaro
```

### Disable RPC Server

Run without exposing any RPC interface:

```bash
eyarod --norpc
```

### Use a SOCKS5 Proxy for P2P Traffic

```bash
eyarod --proxy=127.0.0.1:9050
```

### Advertise a Specific External IP

Useful when the node is behind NAT with a known public IP:

```bash
eyarod --externalip=203.0.113.10
```

### Run with Blocks-Only Mode (No Transaction Relay)

```bash
eyarod --blocksonly
```

### Development Network with Custom DAG Parameters

```bash
eyarod --devnet --override-dag-params-file=/path/to/dag_params.json
```

### Start with LevelDB Backend

```bash
eyarod --dbtype=leveldb
```

## Notes & Caveats

### Disk Space

- A fully synced mainnet node requires significant disk space (tens to hundreds of gigabytes, depending on chain history and whether archival mode is enabled).
- **Pruned nodes** (default): Only recent block data is retained after each pruning point movement. Standard `data-retention-hours` of 0 uses the consensus-defined pruning depth.
- **Archival nodes** (`--archival`): All block data is retained indefinitely. Expect several times the disk usage of a pruned node. Disk usage grows continuously over time.

### Memory

- The UTXO cache defaults to 5 GB (`--maxutxocachesize 5000000000`). This is the amount of UTXO data loaded from disk into RAM.
- The PebbleDB cache uses 2 GiB by default; LevelDB uses 256 MiB.
- The Go runtime's soft memory limit is set to 8 GB by default (`GOMEMLIMIT`).
- For systems with less RAM, reduce `--maxutxocachesize` and consider lowering `GOMEMLIMIT` and `GOGC`.
- For systems with more RAM, increasing `--maxutxocachesize` and the database cache (fixed per backend) can improve sync and query performance.

### Port Requirements

The following ports must be accessible for the node to function properly:

| Port | Protocol | Purpose | Default Network |
|---|---|---|---|
| 42420 | TCP | RPC server (HTTP/WebSocket) | Mainnet |
| 42421 | TCP | P2P networking | Mainnet |
| 42422 | TCP | RPC server | Testnet |
| 42423 | TCP | P2P networking | Testnet |
| 42424 | TCP | RPC server | Simnet |
| 42425 | TCP | P2P networking | Simnet |
| 42426 | TCP | RPC server | Devnet |
| 42427 | TCP | P2P networking | Devnet |

The P2P port must accept incoming connections from other Eiyaro nodes for the node to contribute to the network. If the node is behind NAT, use `--externalip` to advertise the correct public address or `--upnp` to attempt automatic port mapping.

The RPC port only needs to be accessible from trusted clients (wallets, applications).

### IBD (Initial Block Download)

During initial sync, the node downloads all historical headers and blocks from peers. IBD timeouts are enforced:

- Default timeout: **120 minutes** per IBD peer. If a peer fails to provide blocks fast enough, the node disconnects and tries another.
- Near-sync timeout: **10 minutes** when close to the chain tip.
- Minimum performance: **10 headers/second** and **10 blocks/second** per peer (configurable via `--min-headers-per-second` and `--min-blocks-per-second`).

These timeouts can be disabled with `--disable-ibd-timeout` for debugging or slow connections.

### Peer Limits

- Default target outbound peers: **8** (`--outpeers`)
- Default maximum inbound peers: **500** (`--maxinpeers`)
- When `--connect` is used, outbound peering is reduced to 1 and DNS seeding is disabled.

### Database Version Compatibility

The database includes a version check at startup. Switching between incompatible database versions or between networks without resetting may cause startup failures. Use `--reset-db` when switching networks or upgrading across breaking database version changes.

### Network Namespacing

The data directory is automatically namespaced by network name. For example:
- `~/.eyaro/eiyaro-mainnet/` — mainnet data
- `~/.eyaro/eiyaro-testnet/` — testnet data
- `~/.eyaro/eiyaro-simnet/` — simnet data

This means you can run the same binary against multiple networks by specifying different network flags, and each network's data is completely isolated.

## FAQ

### How do I reset the blockchain data?

Stop the node, then either:

1. Delete the data directory manually:
   ```bash
   rm -rf ~/.eyaro/eiyaro-mainnet/datadir2
   ```

2. Or restart with the `--reset-db` flag:
   ```bash
   eyarod --reset-db
   ```
   This deletes the database directory before starting, then begins re-syncing from genesis.

### What ports need to be open?

For a mainnet node operating as a full peer:

- **Inbound P2P** (`42421` by default): Must be open if you want other nodes to connect to you. This is not strictly required for the node to function (it can make outbound connections), but it contributes to network health.
- **RPC** (`42420` by default): Only needs to be accessible from your own applications. In production, bind it to `127.0.0.1` or use a reverse proxy with authentication. Never expose the raw RPC port to the public internet without TLS and authentication.
- For testnet, the corresponding ports are `42423` (P2P) and `42422` (RPC).

### How to check if the node is syncing?

Check the log output. During IBD, eyarod logs messages indicating sync progress:

```
[INF] IBD: Downloaded X headers, Y blocks from peer <address>
```

You can also query the node via RPC:

```bash
# Using a tool like curl against the RPC endpoint
curl -s -X POST -H "Content-Type: application/json" \
  -d '{"method":"getInfo"}' \
  http://localhost:42420/
```

Look for the `isSynced` field in the response — it will be `true` when the node is fully caught up with the network.

### How do I run eyarod as a background service on Linux?

Use systemd. Create a service unit file at `/etc/systemd/system/eyarod.service`:

```ini
[Unit]
Description=Eiyaro Full Node
After=network.target

[Service]
ExecStart=/usr/local/bin/eyarod
User=eyaro
Group=eyaro
Restart=on-failure
LimitNOFILE=2048

[Install]
WantedBy=multi-user.target
```

Then enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable eyarod
sudo systemctl start eyarod
```

On Windows, use the built-in `--service` commands as described in the [Windows Service Mode](#windows-service-mode) section.

### What is the difference between --addpeer and --connect?

- `--addpeer`: Adds specific peers that the node will connect to at startup **in addition to** the normal peer discovery process. The node will also connect to peers discovered via DNS seeds and from other peers.
- `--connect`: Connects **exclusively** to the specified peers. DNS seeding is disabled, outbound peering is set to 1, and the node will not make outbound connections to any other peers.

These two options **cannot be used together**.

### Can I run mainnet and testnet nodes on the same machine?

Yes. Each network uses a separate data directory namespaced by network name. Run two instances with different network flags and (optionally) different ports:

```bash
# Terminal 1: Mainnet
eyarod

# Terminal 2: Testnet with custom directories
eyarod --testnet --appdir=~/.eyaro-testnet
```

### How much disk space does a full node need?

This varies with network activity and configuration:
- A **pruned node** (default) keeps only recent chain data and is substantially smaller.
- An **archival node** (`--archival`) retains all historical data and grows continuously.
- As a rule of thumb, provision at least 100 GB for a pruned mainnet node and 500 GB or more for an archival node. Monitor disk usage over time to plan capacity.

### What does the `--utxoindex` flag do?

It builds and maintains an index mapping addresses to their unspent transaction outputs (UTXOs). This is required for RPC methods that query UTXOs by address, such as `getUtxosByAddresses`. Without this flag, such RPC calls will return errors. Enabling the index increases database size and adds some processing overhead during sync.