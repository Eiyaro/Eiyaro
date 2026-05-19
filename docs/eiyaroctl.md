# Eiyaro CLI Tool (eiyaroctl)

## Overview

**eiyaroctl** is the command-line RPC client for interacting with a local or remote
[eyarod](eyarod.md) node. It communicates via the gRPC-based Eiyaro protocol
over a single TCP connection, sending JSON-RPC-style request messages and
printing formatted JSON responses.

The binary is built from [main.go](file:///d:/Eiyaro/Eiyaro/cmd/eiyaroctl/main.go) and
uses protobuf message types defined in the `protowire` package. Commands are
resolved at runtime via reflection — there is no hard-coded dispatch table.
Use `--list-commands` to see every supported command and its parameter
signature.

---

## Quick Start

```bash
# Check node status
eiyaroctl GetInfo

# Get balance for an address
eiyaroctl GetBalanceByAddress eiyaro:qrf0nqv303g9kcu2nc7zqklq9mqgf4p0mg4h5xnhm8

# Submit a transaction (RpcTransaction JSON required)
eiyaroctl SubmitTransaction '{"version":0,"inputs":[...],"outputs":[...]}' false
```

All responses are returned as pretty-printed JSON. Errors are written to stderr
and cause a non-zero exit.

---

## Connection Configuration

eiyaroctl connects to the eyarod RPC port. The address is resolved as follows:

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--rpcserver` | `-s` | `localhost` | Host or `host:port` of the RPC server |
| `--timeout`   | `-t` | `30`     | Request timeout in seconds |

If no port is given, the default port for the active network is appended:

| Network   | Flag          | Default RPC Port |
|-----------|---------------|-----------------:|
| Mainnet   | _(default)_   | `42420`          |
| Testnet   | `--testnet`   | `42422`          |
| Simnet    | `--simnet`    | `42424`          |
| Devnet    | `--devnet`    | `42426`          |

This logic is implemented in [config.go](file:///d:/Eiyaro/Eiyaro/cmd/eiyaroctl/config.go) via
`NormalizeRPCServerAddress`, which delegates to the active network's `dagconfig.Params.RPCPort`.

**Additional flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--json`   | `-j` | Send a raw JSON request instead of a parsed command |
| `--list-commands` | `-l` | Print all available command signatures and exit |
| `--allow-connection-to-different-versions` | `-a` | Skip the client/server version compatibility check |

### Version Check

By default, on every invocation eiyaroctl sends a `GetInfo` request and compares
the remote `serverVersion` against its own build version. If they differ the
tool exits with an error. This prevents accidentally sending RPC calls to a
node running incompatible software. Use `-a` to disable this check.

### Raw JSON Mode

When `--json` (`-j`) is supplied, the value of that flag is posted directly to
the RPC server as a raw JSON string. This allows calling any RPC method — even
those not exposed as top-level eiyaroctl commands. Example:

```bash
eiyaroctl -j '{"getFeeEstimateRequest":{}}'
```

---

## Available Commands

The command name is derived by stripping the suffix `RequestMessage` from the
protobuf request type. Parameters are positional and appear in the order defined
in the `.proto` schema. Use `-` (a single dash) to skip an optional parameter.

To see the exact parameter list for every command, run:

```bash
eiyaroctl --list-commands
```

All commands are defined in [commands.go](file:///d:/Eiyaro/Eiyaro/cmd/eiyaroctl/commands.go).

### Block & DAG Information

#### GetBlockDagInfo

Returns high-level DAG statistics.

```
eiyaroctl GetBlockDagInfo
```

**Parameters:** none

**Response fields:** `networkName`, `blockCount`, `headerCount`, `tipHashes`,
`difficulty`, `pastMedianTime`, `virtualParentHashes`, `pruningPointHash`,
`virtualDaaScore`

**Example output:**
```json
{
    "getBlockDagInfoResponse": {
        "networkName": "eiyaro-mainnet",
        "blockCount": "1523480",
        "headerCount": "1523480",
        "tipHashes": [
            "a3f1b9c2...",
            "d4e5f678..."
        ],
        "difficulty": 1234567890123.456,
        "pastMedianTime": "1695321400123",
        "virtualParentHashes": ["e7f8a901..."],
        "pruningPointHash": "0000000000...",
        "virtualDaaScore": "1523479"
    }
}
```

#### GetBlock

Retrieves a single block by its hash.

```
eiyaroctl GetBlock [Hash] [IncludeTransactions]
```

| Parameter | Type | Description |
|-----------|------|-------------|
| Hash | string | The block hash (hex-encoded) |
| IncludeTransactions | bool | Whether to populate the `transactions` array |

**Example:**
```bash
eiyaroctl GetBlock a3f1b9c2d4e5f678901234567890abcdef1234567890abcdef1234567890abcdef true
```

#### GetBlockByTransactionId

Finds the block containing a specific transaction.

```
eiyaroctl GetBlockByTransactionId [TransactionId] [IncludeTransactions]
```

| Parameter | Type | Description |
|-----------|------|-------------|
| TransactionId | string | Transaction ID to search for |
| IncludeTransactions | bool | Whether to include transaction data in the block |

#### GetBlocks

Returns blocks (and/or hashes) from a given `lowHash` up to the current virtual tip.

```
eiyaroctl GetBlocks [LowHash] [IncludeBlocks] [IncludeTransactions]
```

| Parameter | Type | Description |
|-----------|------|-------------|
| LowHash | string | Starting block hash (hex) |
| IncludeBlocks | bool | Populate the `blocks` field in the response |
| IncludeTransactions | bool | Populate transaction data inside each block |

#### GetBlockCount

Returns the total number of blocks and headers.

```
eiyaroctl GetBlockCount
```

**Parameters:** none

**Response fields:** `blockCount`, `headerCount`

#### GetHeaders

Returns block headers between a starting hash and the current virtual tip.

```
eiyaroctl GetHeaders [StartHash] [Limit] [IsAscending]
```

| Parameter | Type | Description |
|-----------|------|-------------|
| StartHash | string | Starting block hash (hex) |
| Limit | uint64 | Maximum number of headers to return |
| IsAscending | bool | Sort order: `true` = ascending, `false` = descending |

#### GetSelectedTipHash

Returns the hash of the current virtual's selected parent.

```
eiyaroctl GetSelectedTipHash
```

**Parameters:** none

**Response fields:** `selectedTipHash`

#### GetVirtualSelectedParentBlueScore

Returns the blue score of the virtual's current selected parent.

```
eiyaroctl GetVirtualSelectedParentBlueScore
```

**Parameters:** none

**Response fields:** `blueScore`

#### GetVirtualSelectedParentChainFromBlock

Returns the virtual selected parent chain (added/removed chain blocks)
starting from a given block hash.

```
eiyaroctl GetVirtualSelectedParentChainFromBlock [StartHash] [IncludeAcceptedTransactionIds]
```

| Parameter | Type | Description |
|-----------|------|-------------|
| StartHash | string | Starting block hash |
| IncludeAcceptedTransactionIds | bool | Whether to include accepted transaction IDs per block |

---

### Mining & Block Submission

#### GetBlockTemplate

Requests a block template for mining. Callers are expected to solve the
template's proof-of-work and submit it via `SubmitBlock`.

```
eiyaroctl GetBlockTemplate [PayAddress] [ExtraData]
```

| Parameter | Type | Description |
|-----------|------|-------------|
| PayAddress | string | Address to receive the coinbase reward |
| ExtraData | string | Arbitrary extra data to include in the coinbase |

**Response fields:** `block` (RpcBlock), `isSynced`

The `isSynced` field indicates whether the node considers itself synced with the
network. Mining while unsynced is discouraged because blocks may not be accepted.

#### SubmitBlock

Submits a solved block to the node. The block should be the JSON representation
of an `RpcBlock` as returned by `GetBlockTemplate`.

```
eiyaroctl SubmitBlock [Block] [AllowNonDAABlocks]
```

| Parameter | Type | Description |
|-----------|------|-------------|
| Block | RpcBlock | JSON-encoded block (e.g. `{"header":{...},"transactions":[...]}`) |
| AllowNonDAABlocks | bool | Accept blocks that are not DAA-compliant |

#### EstimateNetworkHashesPerSecond

Estimates the current network hash rate.

```
eiyaroctl EstimateNetworkHashesPerSecond [WindowSize] [StartHash]
```

| Parameter | Type | Description |
|-----------|------|-------------|
| WindowSize | uint32 | Number of blocks to sample |
| StartHash | string | Starting block hash for the estimation window |

---

### Transaction & Mempool

#### SubmitTransaction

Broadcasts a raw transaction to the network.

```
eiyaroctl SubmitTransaction [Transaction] [AllowOrphan]
```

| Parameter | Type | Description |
|-----------|------|-------------|
| Transaction | RpcTransaction | JSON-encoded transaction |
| AllowOrphan | bool | Whether to accept orphan transactions (missing parents) |

**Response fields:** `transactionId` — the transaction ID of the submitted
transaction.

#### GetMempoolEntry

Retrieves a specific transaction from the mempool.

```
eiyaroctl GetMempoolEntry [TxId] [IncludeOrphanPool] [FilterTransactionPool]
```

| Parameter | Type | Description |
|-----------|------|-------------|
| TxId | string | Transaction ID to look up |
| IncludeOrphanPool | bool | Search the orphan pool as well |
| FilterTransactionPool | bool | Apply transaction pool filtering |

**Response fields:** `entry` containing `fee`, `transaction`, `isOrphan`.

#### GetMempoolEntries

Retrieves all transactions currently in the mempool.

```
eiyaroctl GetMempoolEntries [IncludeOrphanPool] [FilterTransactionPool]
```

| Parameter | Type | Description |
|-----------|------|-------------|
| IncludeOrphanPool | bool | Include orphan transactions |
| FilterTransactionPool | bool | Apply transaction pool filtering |

#### GetMempoolEntriesByAddresses

Retrieves mempool entries filtered by addresses, grouped into sending
and receiving transactions per address.

```
eiyaroctl GetMempoolEntriesByAddresses [Addresses] [IncludeOrphanPool] [FilterTransactionPool]
```

| Parameter | Type | Description |
|-----------|------|-------------|
| Addresses | []string | Comma-separated list of addresses |
| IncludeOrphanPool | bool | Include orphan transactions |
| FilterTransactionPool | bool | Apply transaction pool filtering |

---

### Address & Balance

#### GetBalanceByAddress

Returns the total balance (in sompi) of unspent outputs for a single address.

> **Requirement:** The node must have been started with `--utxoindex`.

```
eiyaroctl GetBalanceByAddress [Address]
```

| Parameter | Type | Description |
|-----------|------|-------------|
| Address | string | Eiyaro address string (e.g. `eiyaro:q...`) |

**Response fields:** `balance` (uint64, in sompi — the smallest unit).

**Example:**
```bash
eiyaroctl GetBalanceByAddress eiyaro:qrf0nqv303g9kcu2nc7zqklq9mqgf4p0mg4h5xnhm8
```

#### GetUtxosByAddresses

Returns all UTXOs for one or more addresses.

> **Requirement:** The node must have been started with `--utxoindex`.

```
eiyaroctl GetUtxosByAddresses [Addresses]
```

| Parameter | Type | Description |
|-----------|------|-------------|
| Addresses | []string | Comma-separated list of addresses |

**Response fields:** `entries` — array of `UtxosByAddressesEntry` containing
`address`, `outpoint`, and `utxoEntry`.

#### GetCoinSupply

Returns the circulating supply and maximum supply in sompi.

```
eiyaroctl GetCoinSupply
```

**Parameters:** none

**Response fields:** `maxSompi`, `circulatingSompi`

> **Note:** `maxSompi` is a hard-coded theoretical maximum. The actual supply
> may deviate by up to -5% due to the deflationary emission model and cannot be
> measured exactly.

---

### Network & Node Info

#### GetInfo

Returns general node information.

```
eiyaroctl GetInfo
```

**Parameters:** none

**Response fields:**

| Field | Description |
|-------|-------------|
| `p2pId` | Node's P2P identity string |
| `mempoolSize` | Number of transactions in the mempool |
| `serverVersion` | Node software version string |
| `isUtxoIndexed` | Whether UTXO indexing is enabled |
| `isSynced` | Whether the node considers itself in sync |

#### GetCurrentNetwork

Returns the name of the network the node is operating on.

```
eiyaroctl GetCurrentNetwork
```

**Parameters:** none

**Response fields:** `currentNetwork` (e.g. `"eiyaro-mainnet"`)

---

### Peer Management

#### GetConnectedPeerInfo

Retrieves detailed information about all currently connected peers.

```
eiyaroctl GetConnectedPeerInfo
```

**Parameters:** none

**Response fields:** `infos` — array of `GetConnectedPeerInfoMessage`:

| Field | Description |
|-------|-------------|
| `id` | Peer identity |
| `address` | Peer address (`host:port`) |
| `lastPingDuration` | Duration of last ping/pong exchange (ms) |
| `isOutbound` | Whether this node initiated the connection |
| `timeOffset` | Clock offset between peers (ms) |
| `userAgent` | Peer's user agent string |
| `advertisedProtocolVersion` | Protocol version the peer claims to support |
| `timeConnected` | Unix timestamp of when peer connected |
| `isIbdPeer` | Whether this peer serves IBD (Initial Block Download) data |

#### GetPeerAddresses

Returns known peer addresses and banned addresses.

```
eiyaroctl GetPeerAddresses
```

**Parameters:** none

**Response fields:** `addresses` and `bannedAddresses` — arrays of
`GetPeerAddressesKnownAddressMessage` each containing an `Addr` field.

#### AddPeer

Adds a peer to the node's outgoing connection list. This will cause the node
to attempt connecting to the specified peer.

```
eiyaroctl AddPeer [Address] [IsPermanent]
```

| Parameter | Type | Description |
|-----------|------|-------------|
| Address | string | Peer address (`host:port`) |
| IsPermanent | bool | Keep reconnecting after disconnection |

> **Note:** This command is ignored when the node runs in safe RPC mode
> (`--saferpc`).

#### Ban

Bans an IP address from connecting.

```
eiyaroctl Ban [Ip]
```

| Parameter | Type | Description |
|-----------|------|-------------|
| Ip | string | IP address to ban |

#### Unban

Removes a ban on an IP address.

```
eiyaroctl Unban [Ip]
```

| Parameter | Type | Description |
|-----------|------|-------------|
| Ip | string | IP address to unban |

---

### Finality & Consensus

#### ResolveFinalityConflict

Instructs the node to resolve a finality conflict by re-validating the
specified finality block.

```
eiyaroctl ResolveFinalityConflict [FinalityBlockHash]
```

| Parameter | Type | Description |
|-----------|------|-------------|
| FinalityBlockHash | string | Hash of the finality block to resolve |

---

### Additional RPC Methods (via `--json`)

The following RPC methods are defined in the `.proto` schema but are not
registered as top-level eiyaroctl commands. They can be invoked using the
`--json` (`-j`) flag:

| Method | Description |
|--------|-------------|
| `GetBalancesByAddresses` | Query balances for multiple addresses in a single call |
| `GetSubnetwork` | Retrieve subnetwork information by `subnetworkId` (currently unimplemented) |
| `ShutDown` | Gracefully shut down the eyarod node |
| `GetFeeEstimate` | Get current fee estimation buckets |
| `GetUsableAddresses` | Check which addresses from a list have a usable balance |

**Example — fee estimation via raw JSON:**
```bash
eiyaroctl -j '{"getFeeEstimateRequest":{}}'
```

**Example — graceful shutdown via raw JSON:**
```bash
eiyaroctl -j '{"shutDownRequest":{}}'
```

---

## Usage Examples

### Check Sync Status

```bash
$ eiyaroctl GetInfo
{
    "getInfoResponse": {
        "p2pId": "12D3KooW...",
        "mempoolSize": "142",
        "serverVersion": "1.0.0-abc1234",
        "isUtxoIndexed": true,
        "isSynced": true
    }
}
```

### Check Address Balance

```bash
$ eiyaroctl GetBalanceByAddress eiyaro:qrf0nqv303g9kcu2nc7zqklq9mqgf4p0mg4h5xnhm8
{
    "getBalanceByAddressResponse": {
        "balance": "50000000000"
    }
}
```
The balance is returned in **sompi** (the smallest unit). Divide by 100,000,000
to get the amount in Eiyaro (EYR).

### Get Block Information

```bash
$ eiyaroctl GetBlock a3f1b9c2d4e5f678901234567890abcdef1234567890abcdef1234567890abcdef true
{
    "getBlockResponse": {
        "block": {
            "header": {
                "version": 1,
                "hashMerkleRoot": "...",
                "acceptedIdMerkleRoot": "...",
                "utxoCommitment": "...",
                "timestamp": "1695321400123",
                "bits": "1a2b3c4d",
                "nonce": "1234567890",
                "parents": [...]
            },
            "transactions": [...]
        }
    }
}
```

### Get Multiple Address Balances

```bash
$ eiyaroctl -j '{"getBalancesByAddressesRequest":{"addresses":["eiyaro:qrf0nqv303g9kcu2nc7zqklq9mqgf4p0mg4h5xnhm8","eiyaro:qp..."]}}'
```

### Get Fee Estimate

```bash
$ eiyaroctl -j '{"getFeeEstimateRequest":{}}'
```

The response contains `priorityBucket`, `normalBuckets`, and `lowBuckets`,
each with `feerate` (in sompi per gram) and `estimatedSeconds` for inclusion.

### Check DAG Statistics

```bash
$ eiyaroctl GetBlockDagInfo
```

### List Connected Peers

```bash
$ eiyaroctl GetConnectedPeerInfo
```

### Add a Peer

```bash
$ eiyaroctl AddPeer 192.168.1.100:42421 true
```

### Ban a Misbehaving Peer

```bash
$ eiyaroctl Ban 10.0.0.55
```

---

## Notes & Caveats

### RPC Must Be Enabled

The eyarod node must have its RPC server enabled (this is the default). If the
node was started with `--norpc`, eiyaroctl cannot connect. Verify that the
`rpclisten` configuration in `eiyarod.conf` matches the address you are
connecting to.

### Default Listener

By default, eyarod listens for RPC connections only on `localhost` with the
network-appropriate port. To allow remote connections, configure `rpclisten`
accordingly (e.g. `rpclisten=0.0.0.0:42420`).

### UTXO Index Requirement

Commands that query address balances or UTXOs (`GetBalanceByAddress`,
`GetUtxosByAddresses`, `GetBalancesByAddresses`) require the node to have been
started with the `--utxoindex` flag. Without it, these calls will return an
error.

### Parameter Skipping

Use a single dash `-` to skip an optional parameter. This passes the zero value
for the corresponding protobuf field. For example:

```bash
# Skip IncludeOrphanPool, set FilterTransactionPool to true
eiyaroctl GetMempoolEntries - true
```

### Version Compatibility

By default, eiyaroctl refuses to talk to a node whose version string differs
from its own. Use `-a` to bypass this check when working with development builds
or cross-version testing.

### Timeout

The `--timeout` flag (default 30 seconds) applies to the entire request-response
cycle. Long-running calls (e.g. `GetBlocks` over a wide range) may need a larger
timeout. The maximum allowed timeout is approximately 292 billion years.

### Safe RPC Mode

If eyarod is started with `--saferpc`, commands that modify node state
(`AddPeer`, `SubmitTransaction`, `SubmitBlock`, `Ban`, `Unban`, `ShutDown`)
will be rejected with an error.

### JSON Mode Escaping

When using `--json` on the command line, ensure proper shell escaping of the
JSON string. On most systems, single-quoting the entire JSON payload is
sufficient. For complex nested structures, consider piping from a file.

---

*Documentation generated from source:*
- [main.go](file:///d:/Eiyaro/Eiyaro/cmd/eiyaroctl/main.go)
- [commands.go](file:///d:/Eiyaro/Eiyaro/cmd/eiyaroctl/commands.go)
- [config.go](file:///d:/Eiyaro/Eiyaro/cmd/eiyaroctl/config.go)
- [command_parser.go](file:///d:/Eiyaro/Eiyaro/cmd/eiyaroctl/command_parser.go)
- [rpc.proto](file:///d:/Eiyaro/Eiyaro/infrastructure/network/netadapter/server/grpcserver/protowire/rpc.proto)

---

## FAQ

### How do I check if the node is synced?

Run `eiyaroctl GetInfo` and inspect the `isSynced` field in the response. If `isSynced` is `true`, the node considers itself fully caught up with the network. You can also call `GetBlockDagInfo` and compare `virtualDaaScore` against a public explorer to verify sync depth.

### Why does my RPC command timeout?

The default timeout is 30 seconds (`--timeout` / `-t`). Long-running calls like `GetBlocks` over a wide range may need a larger timeout — increase it with `-t 120` or higher. Also verify that the node's RPC server is reachable and not firewalled. If eyarod was started with `--norpc`, eiyaroctl cannot connect at all.

### How do I get help for a specific command?

Run `eiyaroctl --list-commands` to see every supported command along with its parameter names, types, and positions. There is no per-command help flag — the command list is the authoritative reference. For methods not registered as top-level commands, use `--json` (`-j`) with the raw request format described in the RPC API documentation.

### Can I use eiyaroctl against a remote node?

Yes. By default eyarod listens only on `localhost`, but if the node is configured with `rpclisten=0.0.0.0:42420`, you can connect remotely using `eiyaroctl -s <remote-host>:<port> GetInfo`. Ensure the remote host's firewall allows the RPC port, and be aware that `--saferpc` mode on the server will reject state-modifying commands.

### What do I do if eiyaroctl refuses to connect due to version mismatch?

eiyaroctl compares its own build version against the remote node's `serverVersion` on every invocation. If they differ, it exits with an error. Use the `-a` (`--allow-connection-to-different-versions`) flag to bypass this check. This is useful when working with development builds or when the client and server are on different release schedules.