# Eiyaro Configuration Guide

## Overview

Eiyaro (eyarod) uses a hierarchical configuration system with three layers, applied in the following order of precedence (highest wins):

1. **CLI flags** — command-line arguments passed to `eyarod`
2. **Configuration file** — `eiyarod.conf` in INI format
3. **Hardcoded defaults** — sensible defaults embedded in the binary

The load sequence is:

1. Start with the hardcoded default configuration.
2. Pre-parse the command line to determine the config file path (via `--configfile`/`-C`).
3. If the config file does not exist, a default one is created at the expected path from the embedded template.
4. Parse the config file, overlaying its values on top of defaults.
5. Parse CLI flags again, overlaying them on top of the config file values.

The application data directory is automatically namespaced by network name (e.g., `~/.eyarod/mainnet`, `~/.eyarod/testnet-10`), so you can run multiple networks side-by-side without data conflicts.

### Entry Point

The main entry point lives in [main.go](file:///d:/Eiyaro/Eiyaro/cmd/eyarod/main.go). It calls `app.Start()`, which invokes `config.LoadConfig()`. Before that, `init()` sets Go runtime environment variables:

| Environment Variable | Default | Description |
|----------------------|---------|-------------|
| `GOGC` | `100` | Go garbage collection target percentage |
| `GOMEMLIMIT` | `8000000000` (8 GB) | Soft memory limit for the Go runtime |
| `EYAROD_PROFILER` | _(not set)_ | If set, starts a pprof HTTP server on the port specified by `PROFILER_PORT` (default `8080`) |

---

## Configuration File

The configuration file uses INI format with an `[Application Options]` section. All parameters are optional; omitting a parameter falls back to its default.

By default, eyarod looks for the config file at:
- **POSIX** (Linux/macOS): `~/.eyaro/eiyarod.conf`
- **Windows**: `%LOCALAPPDATA%\eyaro\eiyarod.conf`

Override the path with `--configfile` / `-C`:

```bash
eyarod --configfile=/etc/eyaro/eiyarod.conf
```

Environment variables in the config file are expanded using POSIX `$VARIABLE` syntax. The tilde `~` is expanded to the home directory.

---

## Configuration Parameters

All parameters correspond to fields in the `Flags` struct defined in [config.go](file:///d:/Eiyaro/Eiyaro/infrastructure/config/config.go).

### General / Application

| Parameter | CLI Flag | Type | Default | Description |
|-----------|----------|------|---------|-------------|
| `appdir` | `--appdir` / `-b` | string | `~/.eyaro` (OS-dependent) | Application data directory for blockchain data, logs, and peer cache. Namespaced by network name. |
| `logdir` | `--logdir` | string | `<appdir>/logs` | Directory for log output files. |
| `loglevel` | `--loglevel` / `-d` | string | `info` | Logging level. Valid values: `trace`, `debug`, `info`, `warn`, `error`, `critical`. Also supports per-subsystem levels: `<subsystem>=<level>,...`. Use `show` to list available subsystems. |
| `configfile` | `--configfile` / `-C` | string | `<appdir>/eiyarod.conf` | Path to the configuration file. |
| `version` | `--version` / `-V` | bool | false | Display version information and exit. |
| `profile` | `--profile` | string | _(empty)_ | Enable HTTP pprof profiling on the given port (must be between 1024 and 65535). Accessible at `http://localhost:<port>/debug/pprof`. |
| `uacomment` | `--uacomment` | []string | _(empty)_ | Comment to append to the user agent string advertised to peers (see BIP 14). Characters `/`, `:`, `(`, `)` are forbidden. |

---

### Network Selection

Defined in the `NetworkFlags` struct in [network.go](file:///d:/Eiyaro/Eiyaro/infrastructure/config/network.go). Only one network flag may be specified at a time.

| Parameter | CLI Flag | Type | Default | Description |
|-----------|----------|------|---------|-------------|
| _(none)_ | _(no flag)_ | — | mainnet | If no network flag is given, the node runs on **mainnet**. |
| `testnet` | `--testnet` | bool | false | Use the standard test network (testnet-10). |
| `testnet-b5` | `--testnet-b5` | bool | false | Use testnet-5 (B5) network. |
| `testnet-b10` | `--testnet-b10` | bool | false | Use testnet-10 (B10) network. |
| `simnet` | `--simnet` | bool | false | Use the simulation test network. On simnet, the config file is not auto-created unless `--configfile` is explicitly set. |
| `devnet` | `--devnet` | bool | false | Use the development test network. Allows overriding DAG parameters via `--override-dag-params-file`. |
| `override-dag-params-file` | `--override-dag-params-file` | string | _(empty)_ | Path to a JSON file overriding DAG parameters. Allowed **only on devnet**. See [DAG Parameter Overrides](#dag-parameter-overrides). |

---

### P2P Network

| Parameter | CLI Flag | Type | Default | Description |
|-----------|----------|------|---------|-------------|
| `listen` | `--listen` | []string | `0.0.0.0:<defaultPort>` | Interfaces and ports to listen on for incoming P2P connections. Default port is network-dependent (mainnet: 42421, testnet: 42423). Multiple addresses allowed. |
| `nolisten` | `--nolisten` | bool | false | Disable listening for incoming P2P connections. Automatically enabled when `--connect` or `--proxy` is used without explicit `--listen`. |
| `externalip` | `--externalip` | []string | _(empty)_ | External IP address(es) to advertise to peers. Useful when behind NAT. |
| `connect` | `--connect` | []string | _(empty)_ | Connect **only** to these peers. Disables listening, DNS seeding, and sets `outpeers` to 1. Mutually exclusive with `--addpeer`. |
| `addpeer` | `--addpeer` / `-a` | []string | _(empty)_ | Persistent peers to maintain connections with. Does not restrict discovery of other peers. Mutually exclusive with `--connect`. |
| `outpeers` | `--outpeers` | int | 8 | Target number of outbound peer connections. |
| `maxinpeers` | `--maxinpeers` | int | 500 | Maximum number of inbound peer connections. |
| `enablebanning` | `--enablebanning` | bool | false | Enable automatic banning of misbehaving peers. |
| `banthreshold` | `--banthreshold` | uint32 | 100 | Maximum ban score before a peer is disconnected and banned. |
| `banduration` | `--banduration` | duration | 24h | Duration to ban misbehaving peers. Valid units: `s`, `m`, `h`. Minimum: 1 second. |
| `whitelist` | `--whitelist` | []string | _(empty)_ | IP addresses or CIDR networks exempt from banning. Examples: `192.168.1.0/24`, `::1`. |
| `nodnsseed` | `--nodnsseed` | bool | false | Disable DNS seeding for peer discovery. |
| `dnsseed` | `--dnsseed` | string | _(empty)_ | Override DNS seed hostname (only one hostname allowed). |
| `grpcseed` | `--grpcseed` | string | _(empty)_ | Hostname of a gRPC server for seeding peers. |
| `proxy` | `--proxy` | string | _(empty)_ | SOCKS5 proxy address for P2P connections (e.g., `127.0.0.1:9050`). Disables listening unless `--listen` is explicitly set. |
| `proxyuser` | `--proxyuser` | string | _(empty)_ | Username for proxy authentication. |
| `proxypass` | `--proxypass` | string | _(empty)_ | Password for proxy authentication. Masked in help output. |
| `upnp` | `--upnp` | bool | false | Use UPnP to automatically forward the listening port on supported NAT devices. Has no effect if external IPs are specified. |
| `disallow-loopback-p2p` | `--disallow-loopback-p2p` | bool | false | Disallow outbound P2P connections to loopback addresses (127.0.0.1, ::1, localhost). Prevents accidental self-connections. |
| `nopeerbloomfilters` | `--nopeerbloomfilters` | bool | false | Disable bloom filtering support (see BIP 111). |
| `blocksonly` | `--blocksonly` | bool | false | Reject all transactions from remote peers; only relay blocks. |
| `force-same-version` | `--force-same-version` | bool | false | Disconnect peers whose advertised protocol version does not exactly match this node's version. |

---

### RPC Server

| Parameter | CLI Flag | Type | Default | Description |
|-----------|----------|------|---------|-------------|
| `rpclisten` | `--rpclisten` | []string | `127.0.0.1:<rpcPort>` | Interfaces and ports for the RPC server. Default port is network-dependent (mainnet: 42420, testnet: 16210). |
| `norpc` | `--norpc` | bool | false | Disable the built-in RPC server. |
| `saferpc` | `--saferpc` | bool | false | Disable RPC commands that modify node state (safe/read-only mode). |
| `rpccert` | `--rpccert` | string | `<appdir>/rpc.cert` | Path to the RPC TLS certificate file. |
| `rpckey` | `--rpckey` | string | `<appdir>/rpc.key` | Path to the RPC TLS private key file. |
| `rpcmaxclients` | `--rpcmaxclients` | int | 500 | Maximum number of concurrent standard RPC clients. |
| `rpcmaxwebsockets` | `--rpcmaxwebsockets` | int | 250 | Maximum number of concurrent RPC WebSocket connections. |
| `rpcmaxconcurrentreqs` | `--rpcmaxconcurrentreqs` | int | 10000 | Maximum number of concurrent RPC requests processed simultaneously. Must be >= 0. |

---

### Mining & Block Creation

| Parameter | CLI Flag | Type | Default | Description |
|-----------|----------|------|---------|-------------|
| `blockmaxmass` | `--blockmaxmass` | uint64 | 10000000 | Maximum transaction mass (weight) used when creating a block template. Valid range: 1000 to 10000000. |
| `miningaddr` | _(passed via RPC)_ | string | _(empty)_ | Mining reward address (set via RPC `setMiningAddress`). Target address for coinbase rewards. |

---

### Transaction Relay & Mempool

| Parameter | CLI Flag | Type | Default | Description |
|-----------|----------|------|---------|-------------|
| `minrelaytxfee` | `--minrelaytxfee` | float64 | 0.00001 (1e-5) | Minimum transaction fee in EYR/kB to be considered non-zero. Must be greater than 0. |
| `maxorphantx` | `--maxorphantx` | uint64 | 100 | Maximum number of orphan transactions kept in memory. |
| `relaynonstd` | `--relaynonstd` | bool | false | Relay non-standard transactions regardless of the default network setting. Mutually exclusive with `--rejectnonstd`. |
| `rejectnonstd` | `--rejectnonstd` | bool | false | Reject non-standard transactions regardless of the default network setting. Mutually exclusive with `--relaynonstd`. |

---

### Compound Transaction Rate Limiting

| Parameter | CLI Flag | Type | Default | Description |
|-----------|----------|------|---------|-------------|
| `max-compound-tx-per-minute` | `--max-compound-tx-per-minute` | uint64 | 10 | Maximum compound transactions per address per minute. |
| `compound-tx-ratelimit-window` | `--compound-tx-ratelimit-window` | uint64 | 1 | Rate limit window duration in minutes. |
| `compound-tx-inputs-threshold` | `--compound-tx-inputs-threshold` | uint64 | 21 | Minimum number of inputs to classify a transaction as compound. |

---

### Wallet Freezing

| Parameter | CLI Flag | Type | Default | Description |
|-----------|----------|------|---------|-------------|
| `freeze-address` | `--freeze-address` | []string | _(empty)_ | Freeze an address, preventing its UTXOs from being spent. Can be specified multiple times. |

---

### Memory & Performance

| Parameter | CLI Flag | Type | Default | Description |
|-----------|----------|------|---------|-------------|
| `maxutxocachesize` | `--maxutxocachesize` | uint64 | 5000000000 (5 GB) | Maximum size of UTXO data loaded from disk into RAM, in bytes. |
| `utxoindex` | `--utxoindex` | bool | false | Enable the UTXO index for fast UTXO lookups. Increases memory and disk usage. |
| `archival` | `--archival` | bool | false | Run as an archival node: never delete old block data when the pruning point advances. Warning: significant disk usage. Mutually exclusive with `--data-retention-hours` and `--pruning-interval-hours`. |
| `data-retention-hours` | `--data-retention-hours` | uint64 | 0 | Minimum hours of chain data to retain before allowing pruning deletion. 0 uses standard pruning depth rules. |
| `pruning-interval-hours` | `--pruning-interval-hours` | uint64 | 0 | Minimum hours between UTXO recalculation and block deletion triggers. 0 prunes at every pruning point update. |
| `deletion-depth` | `--deletion-depth` | uint64 | 0 | Multiplies the pruning depth for block deletion. 0 uses the configured pruning depth. *(hidden flag)* |
| `sigcachemaxsize` | `--sigcachemaxsize` | uint | 100000 | Maximum number of entries in the signature verification cache. |
| `dbtype` | `--dbtype` | string | _(auto)_ | Database backend to use for the Block DAG. If not specified, uses the internal default. |
| `utxo-default-max-limit` | `--utxo-default-max-limit` | uint32 | 0 | Default maximum limit for UTXO pull requests. |
| `use-hoohash-c-library` | `--use-hoohash-c-library` | bool | true (Linux arm64), false (otherwise) | Use the hoohash C library for Proof-of-Work calculation on block versions >= 5. |

---

### IBD (Initial Block Download) Tuning

| Parameter | CLI Flag | Type | Default | Description |
|-----------|----------|------|---------|-------------|
| `disable-ibd-timeout` | `--disable-ibd-timeout` | bool | false | Disable IBD timeout entirely. The node will not disconnect slow IBD peers. |
| `ibd-timeout` | `--ibd-timeout` | duration | 120m | Maximum time allowed for IBD before disconnecting the peer and trying another. |
| `nearly-synced-ibd-timeout` | `--nearly-synced-ibd-timeout` | duration | 10m | Maximum time allowed for IBD when nearly synced before reconnecting. |
| `ibd-dequeue-timeout` | `--ibd-dequeue-timeout` | duration | 1m | Maximum time to wait for a block to be dequeued during IBD before disconnecting. |
| `min-headers-per-second` | `--min-headers-per-second` | int | 10 | Minimum headers per second required from an IBD peer before disconnecting. |
| `min-blocks-per-second` | `--min-blocks-per-second` | int | 10 | Minimum blocks per second required from an IBD peer before disconnecting. |

---

### Hidden / Advanced Flags

These flags are marked `hidden:"true"` and are primarily intended for testing and development.

| Parameter | CLI Flag | Type | Default | Description |
|-----------|----------|------|---------|-------------|
| `allow-submit-block-when-not-synced` | `--allow-submit-block-when-not-synced` | bool | false | Accept blocks from RPC while the node is not fully synced. |
| `enable-sanity-check-pruning-utxo` | `--enable-sanity-check-pruning-utxo` | bool | false | When moving the pruning point, verify that the UTXO set matches the UTXO commitment. |
| `protocol-version` | `--protocol-version` | uint32 | 7 | Use a non-default P2P protocol version. |

---

### Windows Service Options

| Parameter | CLI Flag | Type | Default | Description |
|-----------|----------|------|---------|-------------|
| `service` | `--service` / `-s` | string | _(empty)_ | Windows service command. Valid values: `install`, `remove`, `start`, `stop`. Only available on Windows. |

---

### DAG Parameter Overrides

When running on devnet (`--devnet`), you can override core DAG consensus parameters via a JSON file specified with `--override-dag-params-file`. The following fields are supported (all optional — only specified fields override defaults):

| JSON Field | Type | Description |
|------------|------|-------------|
| `k` | KType | DAG confirmation depth K |
| `maxBlockParents` | KType | Maximum number of block parents |
| `mergeSetSizeLimit` | uint64 | Merge set size limit |
| `maxBlockMass` | uint64 | Maximum block mass |
| `maxCoinbasePayloadLength` | uint64 | Maximum coinbase payload length |
| `massPerTxByte` | uint64 | Mass per transaction byte |
| `massPerScriptPubKeyByte` | uint64 | Mass per script public key byte |
| `massPerSigOp` | uint64 | Mass per signature operation |
| `coinbasePayloadScriptPublicKeyMaxLength` | uint8 | Maximum coinbase payload script public key length |
| `powMax` | string (hex) | Maximum Proof-of-Work target (hex-encoded big integer) |
| `blockCoinbaseMaturity` | uint64 | Block coinbase maturity period |
| `subsidyGenesisReward` | uint64 | Genesis block subsidy reward |
| `subsidyPastRewardMultiplier` | float64 | Past reward multiplier |
| `subsidyMergeSetRewardMultiplier` | float64 | Merge set reward multiplier |
| `targetTimePerBlockInMilliSeconds` | int64 | Target time between blocks in milliseconds |
| `finalityDuration` | int64 | Finality duration in milliseconds |
| `timestampDeviationTolerance` | int | Timestamp deviation tolerance in seconds |
| `difficultyAdjustmentWindowSize` | int | Difficulty adjustment window size |
| `relayNonStdTxs` | bool | Relay non-standard transactions |
| `acceptUnroutable` | bool | Accept unroutable addresses |
| `enableNonNativeSubnetworks` | bool | Enable non-native subnetworks |
| `disableDifficultyAdjustment` | bool | Disable difficulty adjustment |
| `skipProofOfWork` | bool | Skip Proof-of-Work validation |
| `hardForkOmitGenesisFromParentsDAAScore` | uint64 | Hard fork: omit genesis from parents DAA score |

---

## Environment Variables

### Runtime Environment

These are set/read in [main.go](file:///d:/Eiyaro/Eiyaro/cmd/eyarod/main.go):

| Variable | Default | Description |
|----------|---------|-------------|
| `GOGC` | `100` | Go GC target percentage. Controls GC frequency. |
| `GOMEMLIMIT` | `8000000000` | Soft memory limit in bytes for the Go runtime (8 GB). |
| `EYAROD_PROFILER` | _(not set)_ | If set to any non-empty value, starts a pprof HTTP server for live profiling. |
| `PROFILER_PORT` | `8080` | Port for the profiler HTTP server (used only when `EYAROD_PROFILER` is set). |

### Database Storage (PebbleDB)

Defined in [pebble/options.go](file:///d:/Eiyaro/Eiyaro/infrastructure/db/database/pebble/options.go):

| Variable | Default | Description |
|----------|---------|-------------|
| `HTND_BLOOM_FILTER_LEVEL` | `16` | Bloom filter bits per key (range: 8–20). |
| `HTND_MEMTABLE_SIZE_MB` | `512` | Memtable size in MB (minimum: 16). |
| `HTND_MEMTABLE_STOP_THRESHOLD` | `8` | Number of memtables before writes are stalled (minimum: 2). |
| `HTND_BASE_FILE_SIZE_MB` | _(auto)_ | Base SST file size in MB (minimum: 16). Auto-calculated as `memtable_size / 2`, clamped to 32–128 MB. |
| `HTND_PEBBLE_CACHE_MB` | `4096` (4 GB) | Block cache size in MB (minimum: 256). |
| `HTND_L0_COMPACTION_THRESHOLD` | `64` | L0 compaction trigger threshold. |
| `HTND_L0_STOP_WRITES_THRESHOLD` | `200` | L0 file count at which writes are stopped. |
| `HTND_L0_COMPACTION_FILE_THRESHOLD` | `64` | L0 file count at which compaction is triggered. |
| `HTND_L0_COMPACTION_CONCURRENCY` | `12` | Number of concurrent L0 compactions. |
| `HTND_PEBBLE_MAX_OPEN_FILES` | `1024` | Maximum number of open files. |
| `HTND_PEBBLE_LOG_EVENTS` | `false` | Enable detailed PebbleDB event logging. |
| `HTND_PEBBLE_LOG_EVENTS_MIN_MS` | `250` | Minimum duration in ms for a PebbleDB event to be logged. |

### Database Storage (LevelDB)

Defined in [ldb/options.go](file:///d:/Eiyaro/Eiyaro/infrastructure/db/database/ldb/options.go):

| Variable | Default | Description |
|----------|---------|-------------|
| `HTND_BLOOM_FILTER_LEVEL` | _(default)_ | LevelDB bloom filter bits per key. Falls back to `BLOOM_FILTER_LEVEL` if not set. |
| `KSDB_COMPACTION_L0_TRIGGER` | `8` | L0 compaction trigger for LevelDB. |
| `KSDB_WRITE_L0_SLOWDOWN` | `24` | L0 slow-down trigger for LevelDB. |
| `KSDB_WRITE_L0_PAUSE` | `48` | L0 pause trigger for LevelDB. |

### Consensus

| Variable | Default | Description |
|----------|---------|-------------|
| `HTND_LARGE_CACHE_DIVISOR` | _(not set)_ | Overrides a large cache divisor for consensus operations. |
| `HTND_PROFILER` | _(not set)_ | Enables consensus-level profiling output. |

### Logging

| Variable | Default | Description |
|----------|---------|-------------|
| `LOGFLAGS` | _(empty)_ | Comma-separated log flags to enable additional logging features. |

### Testing

| Variable | Default | Description |
|----------|---------|-------------|
| `EIYARO_TEST_LOGLEVEL` | _(not set)_ | Log level override for integration tests. |

---

## Configuration Precedence Rules

1. **`--connect` and `--addpeer`** are mutually exclusive. Specifying both causes an error.
2. **`--relaynonstd` and `--rejectnonstd`** are mutually exclusive. Specifying both causes an error.
3. **`--archival` and `--data-retention-hours`** are mutually exclusive — archival mode already keeps all data.
4. **`--archival` and `--pruning-interval-hours`** are mutually exclusive — archival mode does not prune.
5. Using **`--proxy`** or **`--connect`** without explicit `--listen` automatically disables listening.
6. Using **`--connect`** automatically disables DNS seeding and sets outbound peer target to 1.
7. Multiple **network flags** (`--testnet`, `--simnet`, `--devnet`, etc.) cannot be used together.
8. On **simnet**, the default config file is not auto-created unless `--configfile` is explicitly set.
9. The `--override-dag-params-file` flag is only valid when `--devnet` is also set.

---

## Sample Configuration File

The complete embedded sample configuration (`sample-htnd.conf`) is shown below. A default copy is created at `<appdir>/eiyarod.conf` on first run if one does not already exist.

```ini
[Application Options]

; ------------------------------------------------------------------------------
; Data settings
; ------------------------------------------------------------------------------

; The directory to store data such as the block DAG and peer addresses. The
; block DAG takes several GB, so this location must have a lot of free space.
; The default is ~/.eyarod/data on POSIX OSes, $LOCALAPPDATA/eyarod/data on Windows,
; ~/Library/Application Support/eyarod/data on Mac OS, and $home/eyarod/data on
; Plan9. Environment variables are expanded so they may be used. NOTE: Windows
; environment variables are typically %VARIABLE%, but they must be accessed with
; $VARIABLE here. Also, ~ is expanded to $LOCALAPPDATA on Windows.
; datadir=~/.eyarod/data


; ------------------------------------------------------------------------------
; Network settings
; ------------------------------------------------------------------------------

; Use testnet.
; testnet=1

; Connect via a SOCKS5 proxy. NOTE: Specifying a proxy will disable listening
; for incoming connections unless listen addresses are provided via the 'listen'
; option.
; proxy=127.0.0.1:9050
; proxyuser=
; proxypass=

; Use Universal Plug and Play (UPnP) to automatically open the listen port
; and obtain the external IP address from supported devices. NOTE: This option
; will have no effect if external IP addresses are specified.
; upnp=1

; Specify the external IP addresses your node is listening on. One address per
; line. eyarod will not contact 3rd-party sites to obtain external ip addresses.
; This means if you are behind NAT, your node will not be able to advertise a
; reachable address unless you specify it here or enable the 'upnp' option (and
; have a supported device).
; externalip=1.2.3.4
; externalip=2002::1234

; ******************************************************************************
; Summary of 'addpeer' versus 'connect'.
;
; Only one of the following two options, 'addpeer' and 'connect', may be
; specified. Both allow you to specify peers that you want to stay connected
; with, but the behavior is slightly different. By default, eyarod will query DNS
; to find peers to connect to, so unless you have a specific reason such as
; those described below, you probably won't need to modify anything here.
;
; 'addpeer' does not prevent connections to other peers discovered from
; the peers you are connected to and also lets the remote peers know you are
; available so they can notify other peers they can to connect to you. This
; option might be useful if you are having problems finding a node for some
; reason (perhaps due to a firewall).
;
; 'connect', on the other hand, will ONLY connect to the specified peers and
; no others. It also disables listening (unless you explicitly set listen
; addresses via the 'listen' option) and DNS seeding, so you will not be
; advertised as an available peer to the peers you connect to and won't accept
; connections from any other peers. So, the 'connect' option effectively allows
; you to only connect to "trusted" peers.
; ******************************************************************************

; Add persistent peers to connect to as desired. One peer per line.
; You may specify each IP address with or without a port. The default port will
; be added automatically if one is not specified here.
; addpeer=192.168.1.1
; addpeer=10.0.0.2:42421
; addpeer=fe80::1
; addpeer=[fe80::2]:42421

; Add persistent peers that you ONLY want to connect to as desired. One peer
; per line. You may specify each IP address with or without a port. The
; default port will be added automatically if one is not specified here.
; NOTE: Specifying this option has other side effects as described above in
; the 'addpeer' versus 'connect' summary section.
; connect=192.168.1.1
; connect=10.0.0.2:42421
; connect=fe80::1
; connect=[fe80::2]:42421

; Maximum number of inbound and outbound peers.
; maxinpeers=125

; Enable banning of misbehaving peers.
; enablebanning=1

; Maximum allowed ban score before disconnecting and banning misbehaving peers.
; banthreshold=100

; How long to ban misbehaving peers. Valid time units are {s, m, h}.
; Minimum 1s.
; banduration=24h
; banduration=11h30m15s

; Add whitelisted IP networks and IPs. Connected peers whose IP matches a
; whitelist will not have their ban score increased.
; whitelist=127.0.0.1
; whitelist=::1
; whitelist=192.168.0.0/24
; whitelist=fd00::/16

; Disable DNS seeding for peers. By default, when eyarod starts, it will use
; DNS to query for available peers to connect with.
; nodnsseed=1

; Specify the interfaces to listen on. One listen address per line.
; NOTE: The default port is modified by some options such as 'testnet', so it is
; recommended to not specify a port and allow a proper default to be chosen
; unless you have a specific reason to do otherwise.
; All interfaces on default port (this is the default):
;  listen=
; All ipv4 interfaces on default port:
;  listen=0.0.0.0
; All ipv6 interfaces on default port:
;   listen=::
; All interfaces on port 42421:
;   listen=:42421
; All ipv4 interfaces on port 42421:
;   listen=0.0.0.0:42421
; All ipv6 interfaces on port 42421:
;   listen=[::]:42421
; Only ipv4 localhost on port 8333:
;   listen=127.0.0.1:8333
; Only ipv6 localhost on port 8333:
;   listen=[::1]:8333
; Only ipv4 localhost on non-standard port 8336:
;   listen=127.0.0.1:8336
; All interfaces on non-standard port 8336:
;   listen=:8336
; All ipv4 interfaces on non-standard port 8336:
;   listen=0.0.0.0:8336
; All ipv6 interfaces on non-standard port 8336:
;   listen=[::]:8336

; Disable listening for incoming connections. This will override all listeners.
; nolisten=1

; Disable peer bloom filtering. See BIP0111.
; nopeerbloomfilters=1

; Add comments to the user agent that is advertised to peers.
; Must not include characters '/', ':', '(' and ')'.
; uacomment=

; Disconnect peers unless they advertise the exact same HTND version.
; force-same-version=1

; ------------------------------------------------------------------------------
; RPC server options - The following options control the built-in RPC server
; which is used to control and query information from a running eyarod process.
; ------------------------------------------------------------------------------

; Specify the interfaces for the RPC server listen on. One listen address per
; line. NOTE: The default port is modified by some options such as 'testnet',
; so it is recommended to not specify a port and allow a proper default to be
; chosen unless you have a specific reason to do otherwise. By default, the
; RPC server will only listen on localhost for IPv4 and IPv6.
; All interfaces on default port:
;   rpclisten=
; All ipv4 interfaces on default port:
;   rpclisten=0.0.0.0
; All ipv6 interfaces on default port:
;   rpclisten=::
; All interfaces on port 42420:
;   rpclisten=:42420
; All ipv4 interfaces on port 42420:
;   rpclisten=0.0.0.0:42420
; All ipv6 interfaces on port 42420:
;   rpclisten=[::]:42420
; Only ipv4 localhost on port 42420:
;   rpclisten=127.0.0.1:42420
; Only ipv6 localhost on port 42420:
;   rpclisten=[::1]:42420
; Only ipv4 localhost on non-standard port 8337:
;   rpclisten=127.0.0.1:8337
; All interfaces on non-standard port 8337:
;   rpclisten=:8337
; All ipv4 interfaces on non-standard port 8337:
;   rpclisten=0.0.0.0:8337
; All ipv6 interfaces on non-standard port 8337:
;   rpclisten=[::]:8337

; Specify the maximum number of concurrent RPC clients for standard connections.
; rpcmaxclients=10

; Use the following setting to disable the RPC server.
; norpc=1


; ------------------------------------------------------------------------------
; Mempool Settings - The following options
; ------------------------------------------------------------------------------

; Set the minimum transaction fee to be considered a non-zero fee,
; minrelaytxfee=0.00001

; Limit orphan transaction pool to 100 transactions.
; maxorphantx=100

; Do not accept transactions from remote peers.
; blocksonly=1

; Relay non-standard transactions regardless of default network settings.
; relaynonstd=1

; Reject non-standard transactions regardless of default network settings.
; rejectnonstd=1


; ------------------------------------------------------------------------------
; Signature Verification Cache
; ------------------------------------------------------------------------------

; Limit the signature cache to a max of 50000 entries.
; sigcachemaxsize=50000


; ------------------------------------------------------------------------------
; Debug
; ------------------------------------------------------------------------------

; Debug logging level.
; Valid levels are {trace, debug, info, warn, error, critical}
; You may also specify <subsystem>=<level>,<subsystem2>=<level>,... to set
; log level for individual subsystems. Use eyarod --loglevel=show to list
; available subsystems.
; loglevel=info

; The port used to listen for HTTP profile requests. The profile server will
; be disabled if this option is not specified. The profile information can be
; accessed at http://localhost:<profileport>/debug/pprof once running.
; profile=6061
```

---

## Usage Examples

### Production Mainnet Node

A typical mainnet node with default settings. No configuration file is strictly required — defaults work out of the box:

```bash
eyarod
```

With a config file for persistent settings:

```ini
[Application Options]
loglevel=info
listen=0.0.0.0
rpclisten=127.0.0.1
maxinpeers=500
outpeers=8
```

### Development Testnet Node

Run a node on the testnet-10 network with UTXO indexing enabled:

```bash
eyarod --testnet-b10 --utxoindex --loglevel=debug
```

Or via config file:

```ini
[Application Options]
testnet-b10=1
utxoindex=1
loglevel=debug
```

### Mining Node

Run a full node with larger memory allocation on mainnet, then configure the mining address via RPC:

```bash
eyarod --loglevel=info --maxutxocachesize=10000000000
```

Then via RPC:

```
setMiningAddress eiyaro:qp4ad2eh72xc8dtjjyz4llxzq9utn6k26uyl644xxw70wskdfl85zsqj9k4vz true
```

### Archival Node

An archival node that retains the entire blockchain history:

```bash
eyarod --archival --loglevel=info
```

Note: Archival mode requires substantial disk space and cannot be combined with `--data-retention-hours` or `--pruning-interval-hours`.

### Private / Trusted Peer Setup

Connect exclusively to known trusted peers:

```bash
eyarod --connect=10.0.0.5:42421 --connect=10.0.0.6:42421
```

This disables listening, DNS seeding, and connections to any other peers.

### Tor / SOCKS5 Proxy

Route all P2P traffic through Tor:

```ini
[Application Options]
proxy=127.0.0.1:9050
listen=127.0.0.1:42421
```

Note: `--listen` must be explicitly set when using `--proxy`, otherwise listening is disabled.

### Devnet with Custom DAG Parameters

```bash
eyarod --devnet --override-dag-params-file=custom-params.json
```

Example `custom-params.json`:

```json
{
    "k": 18,
    "maxBlockParents": 10,
    "maxBlockMass": 500000,
    "targetTimePerBlockInMilliSeconds": 1000,
    "skipProofOfWork": true
}
```

### Performance-Tuned PebbleDB

For nodes with ample RAM, tune the database for better performance:

```bash
# 8 GB block cache, 1 GB memtable
HTND_PEBBLE_CACHE_MB=8192 HTND_MEMTABLE_SIZE_MB=1024 eyarod
```

### Profiling

Enable live profiling for performance analysis:

```bash
# Built-in profile flag
eyarod --profile=6060

# Or via the high-precision profiler (includes block and mutex profiles)
EYAROD_PROFILER=1 PROFILER_PORT=6060 eyarod
```

---

## Default Ports by Network

| Network | P2P Port | RPC Port |
|---------|----------|----------|
| Mainnet | 42421 | 42420 |
| Testnet (testnet-10) | 42423 | 16210 |
| Testnet B5 | 42521 | 16520 |
| Simnet | 18555 | 18556 |
| Devnet | 16611 | 16610 |

These ports are defined in the `dagconfig.Params` structures for each network and are used as defaults when `--listen` or `--rpclisten` are specified without an explicit port.

---

## Default File Paths

| File | Default Location |
|------|-----------------|
| Config file | `<appdir>/eiyarod.conf` |
| Log output | `<appdir>/logs/eiyarod.log` |
| Error log | `<appdir>/logs/eiyarod_err.log` |
| RPC certificate | `<appdir>/rpc.cert` |
| RPC private key | `<appdir>/rpc.key` |
| Blockchain data | `<appdir>/<network>/datadir2/` |

The application directory (`appdir`) defaults to:
- **Linux/BSD**: `~/.eyaro`
- **macOS**: `~/Library/Application Support/Eiyaro`
- **Windows**: `%LOCALAPPDATA%\eyaro`
- **Plan 9**: `$home/eyaro`

---

## FAQ

### How do I see my current configuration?

Run `eyarod --help` to display all available flags and their defaults. The effective configuration (after merging defaults, config file, and CLI flags) is logged at startup at the `info` log level. You can also inspect the auto-generated config file at `<appdir>/eiyarod.conf` to see which values were persisted from your last run.

### What is the difference between --connect and --addpeer?

`--addpeer` adds persistent peers that the node will always attempt to maintain connections with, while still allowing connections to other discovered peers and accepting inbound connections. `--connect` is more restrictive: it connects **only** to the specified peers, disables DNS seeding, sets the outbound peer target to 1, and disables listening unless `--listen` is explicitly set. Use `--addpeer` for normal operation with trusted peers; use `--connect` for fully private/isolated setups.

### How do I change the data directory location?

Set the `--appdir` / `-b` flag to change the root application data directory. The actual blockchain data lives under `<appdir>/<network>/datadir2/`. For example: `eyarod --appdir=/mnt/blockchain-ssd`. The directory is automatically namespaced by network name, so mainnet and testnet data can coexist under the same parent directory.

### How do I enable mining on my node?

First, ensure the node is running and synced. Then configure a mining address via RPC: `eiyaroctl setMiningAddress <your_eiyaro_address> true`. Start a miner (separate binary or integrated CPU miner) that connects to the node's RPC to fetch block templates and submit solved blocks. The coinbase reward will be sent to the configured mining address.

### Can I change database backends without resyncing?

No. Switching from LevelDB to PebbleDB (or vice versa) requires a full blockchain resync. Set the `--dbtype` flag or use environment variables to select the backend, point to a fresh or empty data directory, and let the node sync from the P2P network. There is no in-place migration tool between the two backends.

### How do I run multiple nodes on the same machine?

Use the `--appdir` and `--listen`/`--rpclisten` flags to give each node a distinct data directory and non-conflicting ports. For example: node 1 uses default ports (42420/42421), node 2 uses `--appdir=~/.eyaro-node2 --listen=:42422 --rpclisten=:42421`. Each node must have its own database directory to avoid file lock conflicts.