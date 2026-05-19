# Eiyaro Miner

## Overview

The Eiyaro miner (`eiyarominer`) is a CPU-based mining client that connects to an eyarod node via JSON-RPC. It fetches block templates, performs NogoPow proof-of-work computation, and submits solved blocks to the network. The miner is designed to be ASIC-resistant through its use of a 256×256 fixed-point matrix multiplication algorithm paired with Blake3 hashing.

## Quick Start

Start mining by connecting to a local eyarod node:

```bash
eiyarominer --rpcserver localhost:42420 --miningaddr eiyaro:...
```

The miner will discover your CPU core count automatically and use all cores by default. You will see output similar to:

```
Number of CPU's found: 8
Threads enabled: 8
Version 0.x.x
```

## How Mining Works

The mining process follows a continuous loop:

1. **Subscribe to block templates**: The miner connects to the eyarod node via JSON-RPC and registers for `NewBlockTemplate` notifications. When a new template is available, the node pushes a notification to the miner.

2. **Receive block template**: The template contains the block header data (including the previous block hash, timestamp, target difficulty in compact form, and DAA score), along with the coinbase transaction pointing to the configured mining address.

3. **Compute proof-of-work**: For each nonce value, the miner:
   - Hashes the block header (prevHash + timestamp + 32 zero bytes + nonce) via Blake3 to produce a seed
   - Multiplies two 256×256 integer matrices (matrix A derived from the previous header hash via xoshiro256++, matrix B derived from the block's DAA score)
   - Hashes the resulting matrix elements through Blake3 to produce the final PoW hash
   - Converts the hash to a big integer and compares it against the target

4. **Submit the block**: When a valid nonce is found (hash < target), the miner constructs the final block with the winning nonce and submits it to the eyarod node via the `SubmitBlock` RPC.

5. **Repeat**: The miner continues with the next nonce (or a new template if one arrives from the node), looking for the next block.

## Command-Line Flags

All flags are parsed by the `go-flags` library. The miner supports the following:

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--version` | `-V` | — | Display version information and exit |
| `--rpcserver` | `-s` | `localhost` | RPC server address to connect to. If no port is specified, the default RPC port for the active network is appended automatically |
| `--miningaddr` | — | *(required)* | Address to receive mining rewards. Must be a valid Bech32 address matching the active network prefix |
| `--numblocks` | `-n` | `0` | Number of blocks to mine before exiting. If omitted (or 0), mines indefinitely until interrupted |
| `--threads` | `-t` | NumCPU | Number of CPU threads to use for mining. Defaults to the number of logical CPUs detected on the system |
| `--mine-when-not-synced` | — | `false` | Continue mining even when the connected eyarod node is not fully synced with the network |
| `--profile` | — | *(empty)* | Enable HTTP profiling server on the specified port. Port must be between 1024 and 65535 |
| `--target-blocks-per-second` | — | `5.0` | Maximum block rate limit. Set to `0` for no limit. The default is 5.0 (2× the target network block rate) |

### Network Flags

Exactly one network may be selected. If none is specified, mainnet is used.

| Flag | Network Name | RPC Port | Address Prefix |
|------|-------------|----------|----------------|
| *(default)* | Mainnet | 42420 | `eiyaro` |
| `--testnet` | Testnet | 42422 | `eiyarotest` |
| `--testnet-b5` | Testnet B5 | 42422 | `eiyarotest` |
| `--testnet-b10` | Testnet B10 | 42422 | `eiyarotest` |
| `--simnet` | Simnet | 42424 | `eiyarosim` |
| `--devnet` | Devnet | 42426 | `eydev` |

The `--override-dag-params-file` flag is available only on devnet and allows overriding DAG parameters via a JSON configuration file.

## Configuration

### Platform-Specific Considerations

**Windows**: The miner uses all available CPU threads by default. Be aware that mining is CPU-intensive and will fully utilize all assigned cores. For systems with shared workloads, consider limiting threads with `--threads`.

**Linux**: On Linux ARM64 (aarch64), the miner uses a native-arm64 build path for the pow package. On all other platforms, the standard Go build path applies.

**Power Management**: Mining at full CPU utilization will increase power consumption and heat output. Ensure adequate cooling, especially on laptops or compact systems.

### Log Files

The miner writes logs to:
- `%APPDATA%\eiyarominer\eiyarominer.log` (trace level — all messages)
- `%APPDATA%\eiyarominer\eiyarominer_err.log` (warn level and above)
- Standard output (info level and above)

Logs are structured with subsystem tag `NPMN` and include trace-level detail in the file logs.

### RPC Timeout

The miner uses a 10-second RPC timeout for all requests to the eyarod node. If a timeout occurs during block submission, the miner will attempt to reconnect automatically.

## Block Template Flow

### Template Manager

The template manager ([templatemanager.go](file:///d:/Eiyaro/Eiyaro/cmd/miner/templatemanager/templatemanager.go)) is a thread-safe singleton that holds the current block template and proof-of-work state:

- **`Get()`** — Returns a shallow copy of the current block template, the pre-computed PoW state, and whether the node is synced. A shallow copy ensures concurrent miners can modify the header (nonce) without corrupting the template.

- **`Set()`** — Accepts a `GetBlockTemplateResponseMessage` from the RPC, converts the RPC block into a domain block, pre-computes the PoW state (including the two 256×256 NogoPow matrices), and stores both under a mutex lock.

### Template Acquisition Cycle

The `templatesLoop` function runs in its own goroutine and follows this pattern:

1. Calls `GetBlockTemplate` RPC immediately on startup to fetch the initial template.
2. After the first template, a 100 ms ticker periodically re-fetches templates.
3. Each time a `NewBlockTemplateNotification` arrives from the node, the ticker is reset and a fresh template is fetched.
4. Templates are always fetched with the mining address and a user agent string (`"eiyarominer-{version}"`).

The 100 ms ticker serves as both a keepalive and a fallback mechanism — even if notifications are missed, the miner will get the latest template within 100 ms.

### Coinbase Transaction

The block template returned by eyarod already includes a properly constructed coinbase transaction that pays to the address specified via `--miningaddr`. The miner does not alter transaction selection — it only fills in the nonce field in the block header.

[Code reference: templatesLoop](file:///d:/Eiyaro/Eiyaro/cmd/miner/mineloop.go#L203-L242)

## Proof-of-Work Algorithm

### NogoPow Overview

The Eiyaro network uses **NogoPow**, a CPU-optimized, ASIC-resistant proof-of-work algorithm. It replaces the legacy Hoohash with a 256×256 fixed-point matrix multiplication scheme.

### Algorithm Structure

The PoW computation consists of the following steps:

#### 1. State Pre-computation (once per template)

When a new block template arrives, the `pow.NewState()` function pre-computes two 256×256 matrices:

- **Matrix A (`nogopowMatA`)**: Generated deterministically from the previous block header hash using a xoshiro256++ PRNG seeded with the hash bytes. Each 64-bit PRNG output fills 16 adjacent matrix cells, with each cell containing a 4-bit value (0–15) converted to fixed-point (left-shifted by 30 bits). This produces a 256×256 matrix of int64 values in Q30.30 fixed-point format.

- **Matrix B (`nogopowMatB`)**: Generated from the block's DAA score. Each cell `(i, j)` is computed as `(blockNumber * (i + j + 1)) % 1000`, then converted to fixed-point.

These matrices are computed once and reused across all nonce iterations.

[Code reference: NewState](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/pow/pow.go#L27-L47)

#### 2. Nonce-Dependent Hash

For each nonce value, the miner computes a Blake3 hash of:

```
prevHeader (32 bytes) + timestamp (8 bytes LE) + zeroes (32 bytes) + nonce (8 bytes LE)
```

This produces a 32-byte seed (`powHash`).

#### 3. Matrix Multiplication

Matrix A and Matrix B are multiplied: `matResult = matA × matB`. This is a 256×256 triple-nested loop using fixed-point arithmetic (each product is right-shifted by 30 bits after multiplication). The work is split across 4 goroutines for parallelism.

#### 4. Final Hash

The result matrix is hashed via Blake3. Every element of the 256×256 result matrix is written in little-endian uint64 format, prefixed by the powHash seed bytes. The output is a 32-byte hash.

#### 5. Difficulty Comparison

The final hash is interpreted as a big integer (with byte order reversed — treated as little-endian). If this value is less than or equal to the target, the nonce is valid and the block can be submitted.

### Key Constants

| Constant | Value | Description |
|----------|-------|-------------|
| `nogopowMatrixSize` | 256 | Matrix dimension (256×256) |
| `nogopowFixedPointShift` | 30 | Fixed-point scaling (Q30.30) |
| `nogopowMatrixModulo` | 1000 | Modulo for block number matrix |

[Code reference: NogoPow constants](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/pow/nogopow_heavy.go#L13-L19)

### Matrix Generation (xoshiro256++)

Matrix A is generated using a xoshiro256++ pseudorandom number generator seeded with the previous block header hash. The 256-bit state is initialized from the 32-byte hash. Each call to `Uint64()` rotates and XORs the state to produce the next random 64-bit value. Each value fills 16 matrix cells (4 bits each, from bits 0–3, 4–7, ..., 56–59, 60–63).

[Code reference: xoshiro256++](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/pow/xoshiro.go)

## Mining Loop Details

The mining loop ([mineloop.go](file:///d:/Eiyaro/Eiyaro/cmd/miner/mineloop.go)) orchestrates the entire mining process through several concurrent goroutines.

### Goroutine Architecture

```
main
 ├── templatesLoop       — fetches block templates via RPC
 ├── blocksLoop × N      — (one per thread) mines blocks continuously
 ├── handleFoundBlock    — submits mined blocks via RPC
 └── logHashRate         — reports hashrate every 10 seconds
```

### Block Rate Limiting

When `--target-blocks-per-second` is non-zero, a dual-ticker rate-limiting mechanism is applied:

- **Block ticker**: Enforces a minimum time between individual blocks (`1 / (targetBlocksPerSecond * windowSize)`).
- **Window ticker**: Enforces a minimum time for every `windowSize` blocks (`windowSize / targetBlocksPerSecond`), where `windowSize` is 10.

This allows short bursts of fast blocks while maintaining the overall average.

### Nonce Search

The `mineNextBlock` function:

1. Starts from a random uint64 nonce (via `crypto/rand`).
2. Increments the nonce by 1 each iteration.
3. Calls `getBlockForMining()` to get the latest template and PoW state.
4. Sets `state.Nonce = nonce`, increments the global `hashesTried` counter atomically.
5. Calls `state.CalculateProofOfWorkValue()` to compute the PoW hash.
6. Compares `powNum` against `state.Target` — if `powNum <= target`, the block is valid.
7. Sets the nonce in the block header, records `block.PoWHash`, and returns the block.

If the nonce space (uint64 range) is exhausted for a template, `getBlockForMining` will eventually return a fresh template (non-blocking 200 ms sleep if no template is ready).

[Code reference: mineNextBlock](file:///d:/Eiyaro/Eiyaro/cmd/miner/mineloop.go#L147-L172)

### Error Handling

- **`ErrTimeout`** (on submit): The miner logs a warning and reconnects to the node.
- **`ErrRouteClosed`**: The route was closed (likely due to reconnection); the miner continues silently.
- **`RejectReasonIsInIBD`**: The node is in Initial Block Download. The miner waits 100 ms and retries.
- **Template fetch errors**: Timeout → reconnect. Route closed → continue. Other errors → propagate via error channel.

[Code reference: handleFoundBlock](file:///d:/Eiyaro/Eiyaro/cmd/miner/mineloop.go#L121-L145)

### Sync Status Check

The `getBlockForMining` function checks `isSynced` from the template manager. If the node is not synced and `--mine-when-not-synced` is `false`, the miner skips the current template and sleeps 200 ms before retrying. It logs a warning every 10th retry.

[Code reference: getBlockForMining](file:///d:/Eiyaro/Eiyaro/cmd/miner/mineloop.go#L174-L201)

### Hashrate Reporting

Every 10 seconds, the `logHashRate` goroutine:

1. Reads the atomic `hashesTried` counter.
2. Computes the rate: `kiloHashes = hashesTried / 1000`, then divides by elapsed seconds.
3. Logs: `Current hash rate is X.XX Khash/s`.
4. Subtracts the sampled hashes from the counter (resetting it for the next interval).

[Code reference: logHashRate](file:///d:/Eiyaro/Eiyaro/cmd/miner/mineloop.go#L105-L119)

## Usage Examples

### Basic Mainnet Mining

```bash
eiyarominer --rpcserver localhost:42420 --miningaddr eiyaro:q...
```

Connect to a local eyarod node on the default mainnet RPC port. The miner uses all CPU cores.

### Testnet Mining

```bash
eiyarominer --testnet --miningaddr eiyarotest:q...
```

Mine on the test network. The RPC port defaults to 42422. Note the testnet address prefix `eiyarotest`.

### Remote Node Mining

```bash
eiyarominer -s 192.168.1.100:42420 --miningaddr eiyaro:q...
```

Connect to a remote eyarod node at a specific IP.

### Limited Thread Mining

```bash
eiyarominer -t 4 --miningaddr eiyaro:q...
```

Use only 4 CPU threads, leaving remaining cores free for other tasks.

### Mining a Fixed Number of Blocks

```bash
eiyarominer -n 100 --miningaddr eiyaro:q...
```

Mine exactly 100 blocks, then exit.

### Devnet Mining

```bash
eiyarominer --devnet --miningaddr eydev:q...
```

Mine on a private development network. RPC port defaults to 42426.

### Simnet Mining

```bash
eiyarominer --simnet --miningaddr eiyarosim:q...
```

Mine on a simulation network (useful for local testing with controlled peers).

### Mining with CPU Profiling

```bash
eiyarominer --profile 6060 --miningaddr eiyaro:q...
```

Enable Go's HTTP pprof server on port 6060. Access profiles at `http://localhost:6060/debug/pprof/`.

### Unthrottled Mining

```bash
eiyarominer --target-blocks-per-second 0 --miningaddr eiyaro:q...
```

Disable the block rate limiter. The miner submits blocks as fast as it finds them.

## Notes & Caveats

- **CPU-only mining**: NogoPow is designed to be ASIC-resistant. GPU mining is not supported. The algorithm runs entirely in pure Go.

- **Power consumption**: Mining at 100% CPU utilization across all cores will significantly increase power draw. On battery-powered devices, this will drain the battery quickly.

- **Requires a running eyarod node**: The miner must connect to an eyarod full node that is synced with the network. The miner itself does not maintain a blockchain.

- **Mining address format**: The `--miningaddr` must be a valid Bech32 address with a prefix matching the active network:
  - Mainnet: `eiyaro:...`
  - Testnet: `eiyarotest:...`
  - Simnet: `eiyarosim:...`
  - Devnet: `eydev:...`

- **Hashrate reporting**: Hashrate is displayed in Khash/s (thousands of hashes per second) every 10 seconds. The counter is reset after each report, so each reading represents the average over the preceding 10 seconds.

- **Logging**: All mining progress is logged. Info-level logs go to stdout, while trace-level logs go to the `eiyarominer.log` file. Check this file for detailed debugging information.

- **Multiple miners**: You can run multiple miner instances connecting to the same eyarod node. Each instance independently searches the nonce space. Due to the random starting nonce, they are unlikely to collide.

- **Block rejection**: If a submitted block is rejected because the node is in IBD (Initial Block Download), the miner waits 100 ms and retries. This is normal during node startup.

## FAQ

### How do I check my hashrate?

The miner automatically reports hashrate every 10 seconds in the log output:

```
Current hash rate is 125.50 Khash/s
```

This represents the average hashrate over the last 10-second window.

### Can I mine without a full node?

No. The miner connects to an eyarod node via JSON-RPC to fetch block templates and submit solved blocks. You must run an eyarod node locally or connect to a remote one that you control.

### Why is my miner not finding blocks?

Common reasons include:

1. **Node not synced**: Check that eyarod is fully synced. If the node is in IBD, blocks will be rejected unless `--mine-when-not-synced` is set.

2. **Incorrect mining address**: Ensure `--miningaddr` uses the correct Bech32 prefix for your network. A mainnet prefix (`eiyaro:`) won't work on testnet.

3. **High network difficulty**: Mining is probabilistic. Even with good hardware, finding a block may take time depending on the current network difficulty.

4. **Connection issues**: Verify the RPC server address and port. The default mainnet port is 42420; testnet is 42422.

### What happens when the node pushes a new block template while I'm mining?

The template manager stores the new template atomically. On the next nonce iteration, `getBlockForMining()` returns the fresh template and the miner begins working on the new block immediately.

### Can I run the miner on a headless server?

Yes. The miner runs as a CLI application with no GUI. It logs to stdout and log files. You can run it in the background or with a process manager like systemd.

### How does the miner handle node disconnections?

If an RPC timeout occurs, the miner calls `client.Reconnect()`, which establishes a new connection to the eyarod node and re-registers for block template notifications. The mining loop continues seamlessly once reconnected.

### Does the miner use GPU acceleration?

No. NogoPow is a pure-CPU algorithm implemented in Go. It is designed to be ASIC-resistant and does not benefit from GPU parallelism beyond what CPU threads provide.