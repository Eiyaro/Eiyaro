# Eiyaro JSON-RPC API Reference

## Overview

Eiyaro exposes a gRPC-based RPC interface that supports JSON-encoded messages via `protojson`. Clients send JSON payloads over HTTP to the RPC endpoint, and the server returns JSON responses.

The wire format wraps each RPC method call in a protocol buffer `EiyaroMessage` envelope. On the wire, the JSON representation uses the **request message name** (without the `Message` suffix) as the top-level key.

### Message Wrapping Pattern

Every request is wrapped in an `EiyaroMessage` envelope. The JSON key for the inner message is the **lowerCamelCase** version of the protobuf message name:

```json
{
  "<lowerCamelCaseMessageName>": {
    <fields>
  }
}
```

Examples:
- `GetBlockRequestMessage` becomes key `getBlockRequest`
- `GetInfoRequestMessage` becomes key `getInfoRequest`

The response is returned with the matching response message key:

```json
{
  "<lowerCamelCaseResponseMessageName>": {
    <fields>
  }
}
```

### RPCError

All response messages may contain an `error` field. If the call succeeded, `error` is omitted or its `message` field is empty. On failure, `error` contains a human-readable message:

```json
{
  "error": {
    "message": "Could not find block with hash ..."
  }
}
```

Always check for the presence of `error` in every response before processing result fields.

---

## Connection

| Parameter        | Value                            |
|------------------|----------------------------------|
| Default endpoint | `http://localhost:42420`         |
| Content-Type     | `application/json`               |
| Protocol         | HTTP POST (gRPC `PostJSON` path) |

---

## Protocol Format Detail

Internally, the Eiyaro RPC server uses gRPC with a singlex `Post` method. The `PostJSON` helper:

1. Parses the incoming JSON into a `protowire.EiyaroMessage` using `protojson.Unmarshal`
2. Deserializes the inner protobuf message into Go `appmessage.Message` structs
3. Dispatches to the matching handler from the registered handler map
4. Converts the response back to protobuf and serializes to JSON via `protojson.Marshal` with `EmitUnpopulated: true`

**Field naming convention**: All JSON field names use **lowerCamelCase** (protojson default). Examples:

| Go struct field       | JSON field name         |
|-----------------------|-------------------------|
| `HashMerkleRoot`      | `hashMerkleRoot`        |
| `IsSynced`            | `isSynced`              |
| `IncludeTransactions` | `includeTransactions`   |
| `VirtualDAAScore`     | `virtualDaaScore`       |
| `SubnetworkID`        | `subnetworkId`          |
| `ScriptPublicKey`     | `scriptPublicKey`       |
| `BlockDAAScore`       | `blockDaaScore`         |
| `TransactionID`       | `transactionId`         |
| `DAAScore`            | `daaScore`              |
| `BlueWork`            | `blueWork`              |
| `UTXOCommitment`      | `utxoCommitment`        |

---

## Data Retrieval Methods

### GetInfo

Returns basic node information including server version, sync status, and mempool size.

**Request:** `GetInfoRequestMessage` — no parameters.

**Request JSON:**
```json
{"getInfoRequest": {}}
```

**Response:** `GetInfoResponseMessage`

| Field          | Type   | Description                              |
|----------------|--------|------------------------------------------|
| `p2pId`        | string | Node's P2P network identity              |
| `mempoolSize`  | uint64 | Number of transactions in the mempool    |
| `serverVersion`| string | Eiyaro server version string             |
| `isUtxoIndexed`| bool   | Whether the UTXO index is enabled        |
| `isSynced`     | bool   | Whether the node is fully synced         |
| `error`        | object | RPCError on failure                      |

**Response JSON:**
```json
{
  "getInfoResponse": {
    "p2pId": "12D3KooW...",
    "mempoolSize": "1523",
    "serverVersion": "1.0.0",
    "isUtxoIndexed": true,
    "isSynced": true
  }
}
```

---

### GetBlockDagInfo

Returns DAG statistics: block count, header count, difficulty, tips, and more.

**Request:** `GetBlockDagInfoRequestMessage` — no parameters.

**Request JSON:**
```json
{"getBlockDagInfoRequest": {}}
```

**Response:** `GetBlockDagInfoResponseMessage`

| Field                | Type     | Description                                        |
|----------------------|----------|----------------------------------------------------|
| `networkName`        | string   | Network name (e.g. `"mainnet"`)                    |
| `blockCount`         | uint64   | Total number of blocks                             |
| `headerCount`        | uint64   | Total number of headers                            |
| `tipHashes`          | []string | Hashes of current DAG tips                         |
| `virtualParentHashes`| []string | Hashes of the virtual block's parents              |
| `difficulty`         | double   | Current network difficulty                         |
| `pastMedianTime`     | int64    | Past median time of the virtual block (Unix ms)    |
| `pruningPointHash`   | string   | Hash of the current pruning point                  |
| `virtualDaaScore`    | uint64   | Virtual block DAA score                            |
| `error`              | object   | RPCError on failure                                |

---

### GetBlock

Returns a block by its hash, optionally including full transaction data.

**Request:** `GetBlockRequestMessage`

| Field                | Type | Description                                     |
|----------------------|------|-------------------------------------------------|
| `hash`               | string | Block hash (hex)                              |
| `includeTransactions`| bool   | Whether to populate the `transactions` array  |

**Request JSON:**
```json
{
  "getBlockRequest": {
    "hash": "abcdef1234567890...",
    "includeTransactions": true
  }
}
```

**Response:** `GetBlockResponseMessage`

| Field   | Type    | Description              |
|---------|---------|--------------------------|
| `block` | RpcBlock | The requested block     |
| `error` | object  | RPCError on failure      |

---

### GetBlocks

Returns blocks between a `lowHash` and the current virtual, optionally including transaction data.

**Request:** `GetBlocksRequestMessage`

| Field                | Type   | Description                                     |
|----------------------|--------|-------------------------------------------------|
| `lowHash`            | string | Starting block hash (hex)                       |
| `includeBlocks`      | bool   | Whether to populate the `blocks` array          |
| `includeTransactions`| bool   | Whether blocks include transaction data         |

**Request JSON:**
```json
{
  "getBlocksRequest": {
    "lowHash": "abcdef1234567890...",
    "includeBlocks": true,
    "includeTransactions": false
  }
}
```

**Response:** `GetBlocksResponseMessage`

| Field        | Type       | Description                               |
|--------------|------------|-------------------------------------------|
| `blockHashes`| []string   | Hashes of blocks from `lowHash` to virtual|
| `blocks`     | []RpcBlock | Block objects (if `includeBlocks=true`)   |
| `error`      | object     | RPCError on failure                       |

---

### GetBlockCount

Returns the current block and header counts. Note: block count may decrease during pruning.

**Request:** `GetBlockCountRequestMessage` — no parameters.

**Request JSON:**
```json
{"getBlockCountRequest": {}}
```

**Response:** `GetBlockCountResponseMessage`

| Field        | Type   | Description               |
|--------------|--------|---------------------------|
| `blockCount` | uint64 | Number of blocks          |
| `headerCount`| uint64 | Number of headers         |
| `error`      | object | RPCError on failure       |

---

### GetHeaders

Returns block headers between a starting hash and the current virtual, up to a limit.

**Request:** `GetHeadersRequestMessage`

| Field        | Type   | Description                                |
|--------------|--------|--------------------------------------------|
| `startHash`  | string | Starting block hash (hex)                  |
| `limit`      | uint64 | Maximum number of headers to return        |
| `isAscending`| bool   | Whether to traverse in ascending order     |

**Request JSON:**
```json
{
  "getHeadersRequest": {
    "startHash": "abcdef1234567890...",
    "limit": "10",
    "isAscending": true
  }
}
```

**Response:** `GetHeadersResponseMessage`

| Field     | Type     | Description          |
|-----------|----------|----------------------|
| `headers` | []string | Block header strings |
| `error`   | object   | RPCError on failure  |

---

### GetSelectedTipHash

Returns the hash of the current virtual block's selected parent.

**Request:** `GetSelectedTipHashRequestMessage` — no parameters.

**Request JSON:**
```json
{"getSelectedTipHashRequest": {}}
```

**Response:** `GetSelectedTipHashResponseMessage`

| Field             | Type   | Description                     |
|-------------------|--------|----------------------------------|
| `selectedTipHash` | string | Selected tip hash (hex)         |
| `error`           | object | RPCError on failure             |

---

### GetCoinSupply

Returns coin supply information in **sompi** (the smallest unit).

**Request:** `GetCoinSupplyRequestMessage` — no parameters.

**Request JSON:**
```json
{"getCoinSupplyRequest": {}}
```

**Response:** `GetCoinSupplyResponseMessage`

| Field              | Type   | Description                                          |
|--------------------|--------|------------------------------------------------------|
| `maxSompi`         | uint64 | Theoretical maximum supply (sompi)                   |
| `circulatingSompi` | uint64 | Current circulating supply (sompi)                   |
| `error`            | object | RPCError on failure                                  |

---

### GetBalanceByAddress

Returns the balance for a single address. Requires `--utxoindex`.

**Request:** `GetBalanceByAddressRequestMessage`

| Field     | Type   | Description                     |
|-----------|--------|---------------------------------|
| `address` | string | Eiyaro address string           |

**Request JSON:**
```json
{
  "getBalanceByAddressRequest": {
    "address": "eiyaro:qz..."
  }
}
```

**Response:** `GetBalanceByAddressResponseMessage`

| Field    | Type   | Description          |
|----------|--------|----------------------|
| `balance`| uint64 | Balance in sompi     |
| `error`  | object | RPCError on failure  |

---

### GetBalancesByAddresses

Returns balances for multiple addresses. Requires `--utxoindex`.

**Request:** `GetBalancesByAddressesRequestMessage`

| Field       | Type     | Description                       |
|-------------|----------|-----------------------------------|
| `addresses` | []string | List of Eiyaro address strings    |

**Request JSON:**
```json
{
  "getBalancesByAddressesRequest": {
    "addresses": ["eiyaro:qz...", "eiyaro:qp..."]
  }
}
```

**Response:** `GetBalancesByAddressesResponseMessage`

| Field     | Type                    | Description          |
|-----------|-------------------------|----------------------|
| `entries` | []BalancesByAddressEntry| Balance entries       |
| `error`   | object                  | RPCError on failure  |

**BalancesByAddressEntry:**

| Field     | Type   | Description          |
|-----------|--------|----------------------|
| `address` | string | Eiyaro address       |
| `balance` | uint64 | Balance in sompi     |

---

### GetUtxosByAddresses

Returns UTXOs for the given addresses. Requires `--utxoindex`.

**Request:** `GetUtxosByAddressesRequestMessage`

| Field       | Type     | Description                                    |
|-------------|----------|------------------------------------------------|
| `addresses` | []string | List of Eiyaro address strings                 |

**Request JSON:**
```json
{
  "getUtxosByAddressesRequest": {
    "addresses": ["eiyaro:qz..."]
  }
}
```

**Response:** `GetUtxosByAddressesResponseMessage`

| Field     | Type                     | Description          |
|-----------|--------------------------|----------------------|
| `entries` | []UtxosByAddressesEntry  | UTXO entries         |
| `error`   | object                   | RPCError on failure  |

**UtxosByAddressesEntry:**

| Field       | Type         | Description                    |
|-------------|--------------|--------------------------------|
| `address`   | string       | Eiyaro address                 |
| `outpoint`  | RpcOutpoint  | Transaction outpoint           |
| `utxoEntry` | RpcUtxoEntry | UTXO entry data                |

---

### GetMempoolEntries

Returns all transactions currently in the mempool.

**Request:** `GetMempoolEntriesRequestMessage`

| Field                  | Type | Description                                |
|------------------------|------|--------------------------------------------|
| `includeOrphanPool`    | bool | Include orphan transactions                |
| `filterTransactionPool`| bool | Filter pending pool (from mempool entries by addresses) |

**Request JSON:**
```json
{
  "getMempoolEntriesRequest": {
    "includeOrphanPool": false,
    "filterTransactionPool": false
  }
}
```

**Response:** `GetMempoolEntriesResponseMessage`

| Field     | Type           | Description               |
|-----------|----------------|---------------------------|
| `entries` | []MempoolEntry | Mempool transaction entries |
| `error`   | object         | RPCError on failure       |

**MempoolEntry:**

| Field         | Type           | Description                      |
|---------------|----------------|----------------------------------|
| `fee`         | uint64         | Transaction fee in sompi         |
| `transaction` | RpcTransaction | The transaction object           |
| `isOrphan`    | bool           | Whether this is an orphan tx     |

---

### GetMempoolEntry

Returns a specific mempool entry by transaction ID.

**Request:** `GetMempoolEntryRequestMessage`

| Field                  | Type   | Description                                |
|------------------------|--------|--------------------------------------------|
| `txId`                 | string | Transaction ID (hex)                       |
| `includeOrphanPool`    | bool   | Include orphan pool in search              |
| `filterTransactionPool`| bool   | Filter pending pool (from mempool entries by addresses) |

**Request JSON:**
```json
{
  "getMempoolEntryRequest": {
    "txId": "abcdef1234567890...",
    "includeOrphanPool": false,
    "filterTransactionPool": false
  }
}
```

**Response:** `GetMempoolEntryResponseMessage`

| Field   | Type         | Description               |
|---------|--------------|---------------------------|
| `entry` | MempoolEntry | The mempool entry         |
| `error` | object       | RPCError on failure       |

---

### GetConnectedPeerInfo

Returns information about all currently connected P2P peers.

**Request:** `GetConnectedPeerInfoRequestMessage` — no parameters.

**Request JSON:**
```json
{"getConnectedPeerInfoRequest": {}}
```

**Response:** `GetConnectedPeerInfoResponseMessage`

| Field   | Type                        | Description          |
|---------|-----------------------------|----------------------|
| `infos` | []GetConnectedPeerInfoMessage| Connected peers      |
| `error` | object                      | RPCError on failure  |

**GetConnectedPeerInfoMessage:**

| Field                      | Type   | Description                                    |
|----------------------------|--------|------------------------------------------------|
| `id`                       | string | Peer's P2P identity                            |
| `address`                  | string | Peer's network address (IP:port)               |
| `lastPingDuration`         | int64  | Last ping/pong round trip time (milliseconds)  |
| `isOutbound`               | bool   | Whether this node initiated the connection     |
| `timeOffset`               | int64  | Peer's reported clock offset (milliseconds)    |
| `userAgent`                | string | Peer's user agent string                       |
| `advertisedProtocolVersion`| uint32 | Protocol version advertised by peer           |
| `timeConnected`            | int64  | Unix timestamp (ms) when peer connected        |
| `isIbdPeer`                | bool   | Whether this peer is the IBD sync peer         |

---

### GetPeerAddresses

Returns known peer addresses and banned peer addresses.

**Request:** `GetPeerAddressesRequestMessage` — no parameters.

**Request JSON:**
```json
{"getPeerAddressesRequest": {}}
```

**Response:** `GetPeerAddressesResponseMessage`

| Field             | Type                                  | Description               |
|-------------------|---------------------------------------|---------------------------|
| `addresses`       | []GetPeerAddressesKnownAddressMessage | Known peer addresses      |
| `bannedAddresses` | []GetPeerAddressesKnownAddressMessage | Banned peer addresses     |
| `error`           | object                                | RPCError on failure       |

**GetPeerAddressesKnownAddressMessage:**

| Field  | Type   | Description                    |
|--------|--------|--------------------------------|
| `Addr` | string | Peer address string (IP:port)  |

> **Note:** The field name `Addr` is uppercase as defined in the protobuf schema.

---

### GetSubnetwork

Returns the gas limit for a specific subnetwork.

**Request:** `GetSubnetworkRequestMessage`

| Field          | Type   | Description                    |
|----------------|--------|--------------------------------|
| `subnetworkId` | string | Subnetwork registry ID (hex)   |

**Request JSON:**
```json
{
  "getSubnetworkRequest": {
    "subnetworkId": "0000000000000000000000000000000000000000000000000000000000000000"
  }
}
```

**Response:** `GetSubnetworkResponseMessage`

| Field      | Type   | Description               |
|------------|--------|---------------------------|
| `gasLimit` | uint64 | Subnetwork gas limit      |
| `error`    | object | RPCError on failure       |

---

### GetBlockTemplate

Requests a block template for mining. Callers are expected to solve the template's proof-of-work and submit it via `SubmitBlock`.

**Request:** `GetBlockTemplateRequestMessage`

| Field        | Type   | Description                                                |
|--------------|--------|------------------------------------------------------------|
| `payAddress` | string | Eiyaro address to receive the coinbase reward              |
| `extraData`  | string | Optional extra data to embed in the coinbase transaction   |

**Request JSON:**
```json
{
  "getBlockTemplateRequest": {
    "payAddress": "eiyaro:qz...",
    "extraData": ""
  }
}
```

**Response:** `GetBlockTemplateResponseMessage`

| Field      | Type     | Description                                                    |
|------------|----------|----------------------------------------------------------------|
| `block`    | RpcBlock | Block template to solve                                         |
| `isSynced` | bool     | Whether the node is synced (unsynced blocks may be rejected)    |
| `error`    | object   | RPCError on failure                                            |

---

### GetFeeEstimate

Returns current fee estimates for transaction prioritization.

**Request:** `GetFeeEstimateRequestMessage` — no parameters.

**Request JSON:**
```json
{"getFeeEstimateRequest": {}}
```

**Response:** `GetFeeEstimateResponseMessage`

| Field      | Type           | Description          |
|------------|----------------|----------------------|
| `estimate` | RPCFeeEstimate | Fee estimate data    |
| `error`    | object         | RPCError on failure  |

**RPCFeeEstimate:**

| Field            | Type               | Description                           |
|------------------|--------------------|---------------------------------------|
| `priorityBucket` | RPCFeerateBucket   | High-priority fee bucket              |
| `normalBuckets`  | []RPCFeerateBucket | Normal-priority fee buckets           |
| `lowBuckets`     | []RPCFeerateBucket | Low-priority fee buckets              |

**RPCFeerateBucket:**

| Field              | Type   | Description                                     |
|--------------------|--------|-------------------------------------------------|
| `feerate`          | double | Fee rate in sompi/gram                           |
| `estimatedSeconds` | double | Estimated confirmation time in seconds          |

---

### GetCurrentNetwork

Returns the name of the network the node is running on.

**Request:** `GetCurrentNetworkRequestMessage` — no parameters.

**Request JSON:**
```json
{"getCurrentNetworkRequest": {}}
```

**Response:** `GetCurrentNetworkResponseMessage`

| Field            | Type   | Description                                           |
|------------------|--------|-------------------------------------------------------|
| `currentNetwork` | string | `"mainnet"`, `"testnet"`, `"devnet"`, or `"simnet"`  |
| `error`          | object | RPCError on failure                                   |

---

### GetBlockByTransactionID

Returns the block that contains the given transaction.

**Request:** `GetBlockByTransactionIDRequestMessage`

| Field                | Type   | Description                                  |
|----------------------|--------|----------------------------------------------|
| `transactionId`      | string | Transaction ID (hex) to look up              |
| `includeTransactions`| bool   | Whether to populate block transaction data    |

**Request JSON:**
```json
{
  "getBlockByTransactionIDRequest": {
    "transactionId": "abcdef1234567890...",
    "includeTransactions": true
  }
}
```

**Response:** `GetBlockByTransactionIDResponseMessage`

| Field   | Type     | Description               |
|---------|----------|---------------------------|
| `block` | RpcBlock | Block containing the tx   |
| `error` | object   | RPCError on failure       |

---

### GetUsableAddresses

Filters a list of addresses and returns only those that exist in the UTXO index. Requires `--utxoindex`.

**Request:** `GetUsableAddressesRequestMessage`

| Field       | Type     | Description                     |
|-------------|----------|---------------------------------|
| `addresses` | []string | List of addresses to filter     |

**Request JSON:**
```json
{
  "getUsableAddressesRequest": {
    "addresses": ["eiyaro:qz...", "eiyaro:qp..."]
  }
}
```

**Response:** `GetUsableAddressesResponseMessage`

| Field       | Type     | Description                               |
|-------------|----------|-------------------------------------------|
| `addresses` | []string | Addresses found in the UTXO index         |
| `error`     | object   | RPCError on failure                       |

---

### GetVirtualSelectedParentChainFromBlock

Returns the virtual selected parent chain from a given `startHash` to the current virtual.

**Request:** `GetVirtualSelectedParentChainFromBlockRequestMessage`

| Field                          | Type   | Description                                         |
|--------------------------------|--------|-----------------------------------------------------|
| `startHash`                    | string | Starting block hash (hex)                           |
| `includeAcceptedTransactionIds`| bool   | Whether to include accepted transaction ID lists     |

**Request JSON:**
```json
{
  "getVirtualSelectedParentChainFromBlockRequest": {
    "startHash": "abcdef1234567890...",
    "includeAcceptedTransactionIds": true
  }
}
```

**Response:** `GetVirtualSelectedParentChainFromBlockResponseMessage`

| Field                    | Type                    | Description                                              |
|--------------------------|-------------------------|----------------------------------------------------------|
| `removedChainBlockHashes`| []string                | Chain blocks removed, high-to-low order                  |
| `addedChainBlockHashes`  | []string                | Chain blocks added, low-to-high order                    |
| `acceptedTransactionIds` | []AcceptedTransactionIds| Transaction IDs accepted per block (if requested)         |
| `error`                  | object                  | RPCError on failure                                      |

**AcceptedTransactionIds:**

| Field                    | Type     | Description                           |
|--------------------------|----------|---------------------------------------|
| `acceptingBlockHash`     | string   | Hash of the accepting block           |
| `acceptedTransactionIds` | []string | IDs of accepted transactions          |

---

## Transaction Methods

### SubmitTransaction

Submits a fully-formed transaction to the mempool for processing and propagation.

**Request:** `SubmitTransactionRequestMessage`

| Field         | Type           | Description                                   |
|---------------|----------------|-----------------------------------------------|
| `transaction` | RpcTransaction | The transaction to submit                     |
| `allowOrphan` | bool           | Whether to allow orphan transactions          |

**Request JSON:**
```json
{
  "submitTransactionRequest": {
    "transaction": {
      "version": 1,
      "inputs": [
        {
          "previousOutpoint": {
            "transactionId": "abcdef...",
            "index": 0
          },
          "signatureScript": "",
          "sequence": 0,
          "sigOpCount": 1
        }
      ],
      "outputs": [
        {
          "amount": "100000000",
          "scriptPublicKey": {
            "version": 0,
            "scriptPublicKey": "76a914..."
          }
        }
      ],
      "lockTime": "0",
      "subnetworkId": "0000000000000000000000000000000000000000000000000000000000000000",
      "gas": "0",
      "payload": ""
    },
    "allowOrphan": false
  }
}
```

**Response:** `SubmitTransactionResponseMessage`

| Field           | Type   | Description                          |
|-----------------|--------|--------------------------------------|
| `transactionId` | string | Transaction ID of submitted tx       |
| `error`         | object | RPCError on failure                  |

---

### SubmitTransactionReplacement

Submits a replacement transaction (RBF) to the mempool.

**Request:** `SubmitTransactionReplacementRequestMessage`

| Field         | Type           | Description                               |
|---------------|----------------|-------------------------------------------|
| `transaction` | RpcTransaction | The replacement transaction               |

**Request JSON:**
```json
{
  "submitTransactionReplacementRequest": {
    "transaction": {
      "version": 1,
      "inputs": [...],
      "outputs": [...],
      "lockTime": "0",
      "subnetworkId": "0000000000000000000000000000000000000000000000000000000000000000",
      "gas": "0",
      "payload": ""
    }
  }
}
```

**Response:** `SubmitTransactionReplacementResponseMessage`

| Field                | Type           | Description                                       |
|----------------------|----------------|---------------------------------------------------|
| `transactionId`      | string         | Transaction ID of the replacement                 |
| `replacedTransaction`| RpcTransaction | The transaction that was replaced (if applicable) |
| `error`              | object         | RPCError on failure                               |

---

## Block Methods

### SubmitBlock

Submits a mined block to the network. Blocks are expected to be generated using `GetBlockTemplate`.

**Request:** `SubmitBlockRequestMessage`

| Field              | Type     | Description                                             |
|--------------------|----------|---------------------------------------------------------|
| `block`            | RpcBlock | The solved block to submit                              |
| `allowNonDAABlocks`| bool     | Whether to allow blocks that don't meet DAA difficulty  |

**Request JSON:**
```json
{
  "submitBlockRequest": {
    "block": {
      "header": {
        "version": 1,
        "parents": [
          {"parentHashes": ["abcdef..."]}
        ],
        "hashMerkleRoot": "...",
        "acceptedIdMerkleRoot": "...",
        "utxoCommitment": "...",
        "timestamp": "1680000000000",
        "bits": "12345678",
        "nonce": "98765",
        "daaScore": "100",
        "blueScore": "100",
        "blueWork": "...",
        "pruningPoint": "..."
      },
      "transactions": [...],
      "verboseData": {
        "hash": "abcdef...",
        "difficulty": 1.5,
        "selectedParentHash": "...",
        "transactionIds": [...],
        "isHeaderOnly": false,
        "blueScore": "100",
        "childrenHashes": [],
        "mergeSetBluesHashes": [],
        "mergeSetRedsHashes": [],
        "isChainBlock": true
      }
    },
    "allowNonDAABlocks": false
  }
}
```

**Response:** `SubmitBlockResponseMessage`

| Field          | Type   | Description                           |
|----------------|--------|---------------------------------------|
| `rejectReason` | enum   | Reason the block was rejected (if any)|
| `error`        | object | RPCError on failure                   |

**RejectReason enum values:**

| Value | Name            | Description              |
|-------|-----------------|--------------------------|
| 0     | `NONE`          | Block accepted           |
| 1     | `BLOCK_INVALID` | Block is invalid         |
| 2     | `IS_IN_IBD`     | Node is in initial block download |

---

## Peer Management

### AddPeer

Adds a persistent outgoing peer to the node's connection list.

**Request:** `AddPeerRequestMessage`

| Field         | Type   | Description                                                 |
|---------------|--------|-------------------------------------------------------------|
| `address`     | string | Peer address in `host:port` format                          |
| `isPermanent` | bool   | Whether to keep reconnecting after disconnection            |

**Request JSON:**
```json
{
  "addPeerRequest": {
    "address": "192.168.1.100:42420",
    "isPermanent": true
  }
}
```

**Response:** `AddPeerResponseMessage`

| Field   | Type   | Description          |
|---------|--------|----------------------|
| `error` | object | RPCError on failure  |

---

### Ban

Bans an IP address from connecting to this node.

**Request:** `BanRequestMessage`

| Field | Type   | Description                |
|-------|--------|----------------------------|
| `ip`  | string | IP address to ban          |

**Request JSON:**
```json
{
  "banRequest": {
    "ip": "192.168.1.100"
  }
}
```

**Response:** `BanResponseMessage`

| Field   | Type   | Description          |
|---------|--------|----------------------|
| `error` | object | RPCError on failure  |

---

### Unban

Removes an IP address from the ban list.

**Request:** `UnbanRequestMessage`

| Field | Type   | Description                  |
|-------|--------|------------------------------|
| `ip`  | string | IP address to unban          |

**Request JSON:**
```json
{
  "unbanRequest": {
    "ip": "192.168.1.100"
  }
}
```

**Response:** `UnbanResponseMessage`

| Field   | Type   | Description          |
|---------|--------|----------------------|
| `error` | object | RPCError on failure  |

---

## Notification Methods (Subscription)

Eiyaro supports push notifications via the RPC connection. To subscribe to notifications, send a `Notify*RequestMessage` over the RPC connection. The server will begin pushing notification messages as events occur on the same connection.

Notifications are sent asynchronously — the client must continuously read from the connection to receive them.

### BlockAddedNotification

Receive notifications whenever a new block is added to the DAG.

**Subscribe:** `NotifyBlockAddedRequestMessage` — no parameters.

```json
{"notifyBlockAddedRequest": {}}
```

**Acknowledge:** `NotifyBlockAddedResponseMessage` (contains `error` or empty on success).

**Notification:** `BlockAddedNotificationMessage`

| Field   | Type     | Description              |
|---------|----------|--------------------------|
| `block` | RpcBlock | The newly added block    |

---

### VirtualSelectedParentChainChangedNotification

Receive notifications when the DAG's virtual selected parent chain changes (reorg).

**Subscribe:** `NotifyVirtualSelectedParentChainChangedRequestMessage`

| Field                          | Type | Description                                    |
|--------------------------------|------|------------------------------------------------|
| `includeAcceptedTransactionIds`| bool | Include accepted transaction IDs in notification|

```json
{
  "notifyVirtualSelectedParentChainChangedRequest": {
    "includeAcceptedTransactionIds": true
  }
}
```

**Acknowledge:** `NotifyVirtualSelectedParentChainChangedResponseMessage` (contains `error` or empty on success).

**Notification:** `VirtualSelectedParentChainChangedNotificationMessage`

| Field                      | Type                    | Description                                         |
|----------------------------|-------------------------|-----------------------------------------------------|
| `removedChainBlockHashes`  | []string                | Chain blocks removed, high-to-low order             |
| `addedChainBlockHashes`    | []string                | Chain blocks added, low-to-high order               |
| `acceptedTransactionIds`   | []AcceptedTransactionIds| Transaction IDs accepted per block (if requested)    |

---

### FinalityConflictNotification / FinalityConflictResolvedNotification

Receive notifications about finality conflicts and their resolution.

**Subscribe:** `NotifyFinalityConflictsRequestMessage` — no parameters.

```json
{"notifyFinalityConflictsRequest": {}}
```

**Acknowledge:** `NotifyFinalityConflictsResponseMessage` (contains `error` or empty on success).

**Notification (conflict):** `FinalityConflictNotificationMessage`

| Field                | Type   | Description                                  |
|----------------------|--------|----------------------------------------------|
| `violatingBlockHash` | string | Hash of the block violating finality rules   |

**Notification (resolved):** `FinalityConflictResolvedNotificationMessage`

| Field                | Type   | Description                                |
|----------------------|--------|--------------------------------------------|
| `finalityBlockHash`  | string | Hash of the resolved finality block        |

---

### UtxosChangedNotification

Receive real-time notifications when UTXOs change for registered addresses. Requires `--utxoindex`.

**Subscribe:** `NotifyUtxosChangedRequestMessage`

| Field       | Type     | Description                                           |
|-------------|----------|-------------------------------------------------------|
| `addresses` | []string | Addresses to watch. Empty means watch all addresses.  |

```json
{
  "notifyUtxosChangedRequest": {
    "addresses": ["eiyaro:qz..."]
  }
}
```

**Acknowledge:** `NotifyUtxosChangedResponseMessage` (contains `error` or empty on success).

**Notification:** `UtxosChangedNotificationMessage`

| Field     | Type                    | Description              |
|-----------|-------------------------|--------------------------|
| `added`   | []UtxosByAddressesEntry | Newly created UTXOs      |
| `removed` | []UtxosByAddressesEntry | Removed (spent) UTXOs   |

**Unsubscribe:** `StopNotifyingUtxosChangedRequestMessage`

| Field       | Type     | Description                             |
|-------------|----------|-----------------------------------------|
| `addresses` | []string | Addresses to stop watching              |

```json
{
  "stopNotifyingUtxosChangedRequest": {
    "addresses": ["eiyaro:qz..."]
  }
}
```

**Unsubscribe Acknowledge:** `StopNotifyingUtxosChangedResponseMessage`

---

### NewBlockTemplateNotification

Receive notifications when a new block template is available for mining.

**Subscribe:** `NotifyNewBlockTemplateRequestMessage` — no parameters.

```json
{"notifyNewBlockTemplateRequest": {}}
```

**Acknowledge:** `NotifyNewBlockTemplateResponseMessage` (contains `error` or empty on success).

**Notification:** `NewBlockTemplateNotificationMessage` — empty message. After receiving this notification, call `GetBlockTemplate` to fetch the new template.

---

## System Methods

### ResolveFinalityConflict

Attempts to resolve a finality conflict by re-orging to the given block.

**Request:** `ResolveFinalityConflictRequestMessage`

| Field               | Type   | Description                                    |
|---------------------|--------|------------------------------------------------|
| `finalityBlockHash` | string | Hash of the block to resolve the conflict to   |

**Request JSON:**
```json
{
  "resolveFinalityConflictRequest": {
    "finalityBlockHash": "abcdef1234567890..."
  }
}
```

**Response:** `ResolveFinalityConflictResponseMessage`

| Field   | Type   | Description          |
|---------|--------|----------------------|
| `error` | object | RPCError on failure  |

---

### ShutDown

Gracefully shuts down the Eiyaro node.

**Request:** `ShutDownRequestMessage` — no parameters.

**Request JSON:**
```json
{"shutDownRequest": {}}
```

**Response:** `ShutDownResponseMessage`

| Field   | Type   | Description          |
|---------|--------|----------------------|
| `error` | object | RPCError on failure  |

---

## Common Types

### RpcBlock

| Field         | Type               | Description                                  |
|---------------|--------------------|----------------------------------------------|
| `header`      | RpcBlockHeader     | Block header                                 |
| `transactions`| []RpcTransaction   | Block transactions (when requested)           |
| `verboseData` | RpcBlockVerboseData| Verbose block metadata                       |

### RpcBlockHeader

| Field                  | Type                  | Description                              |
|------------------------|-----------------------|------------------------------------------|
| `version`              | uint32                | Block protocol version                   |
| `parents`              | []RpcBlockLevelParents| Parent blocks per level                  |
| `hashMerkleRoot`       | string                | Merkle root of transaction hashes        |
| `acceptedIdMerkleRoot` | string                | Merkle root of accepted transaction IDs  |
| `utxoCommitment`       | string                | UTXO set commitment hash                 |
| `timestamp`            | int64                 | Block timestamp (Unix milliseconds)      |
| `bits`                 | uint32                | Compact difficulty target                |
| `nonce`                | uint64                | Nonce used to solve the PoW              |
| `daaScore`             | uint64                | Difficulty Adjustment Algorithm score    |
| `blueScore`            | uint64                | Blue score of the block                  |
| `blueWork`             | string                | Cumulative blue work (hex big integer)   |
| `pruningPoint`         | string                | Pruning point hash                       |

### RpcBlockLevelParents

| Field          | Type     | Description                                   |
|----------------|----------|-----------------------------------------------|
| `parentHashes` | []string | Parent block hashes at this block level       |

### RpcBlockVerboseData

| Field                 | Type     | Description                               |
|-----------------------|----------|-------------------------------------------|
| `hash`                | string   | Block hash (hex)                          |
| `difficulty`          | double   | Block difficulty                          |
| `selectedParentHash`  | string   | Selected parent block hash                |
| `transactionIds`      | []string | Transaction IDs in this block             |
| `isHeaderOnly`        | bool     | Whether block has header-only data        |
| `blueScore`           | uint64   | Blue score of the block                   |
| `childrenHashes`      | []string | Hashes of child blocks                    |
| `mergeSetBluesHashes` | []string | Blue blocks in the merge set              |
| `mergeSetRedsHashes`  | []string | Red blocks in the merge set               |
| `isChainBlock`        | bool     | Whether block is on the selected chain    |

### RpcTransaction

| Field          | Type                         | Description                                 |
|----------------|------------------------------|---------------------------------------------|
| `version`      | uint32                       | Transaction protocol version                |
| `inputs`       | []RpcTransactionInput        | Transaction inputs                          |
| `outputs`      | []RpcTransactionOutput       | Transaction outputs                         |
| `lockTime`     | uint64                       | Lock time (block DAA score or timestamp)    |
| `subnetworkId` | string                       | Subnetwork ID (hex)                         |
| `gas`          | uint64                       | Gas limit for subnetwork transactions       |
| `payload`      | string                       | Transaction payload (hex-encoded)           |
| `verboseData`  | RpcTransactionVerboseData    | Verbose transaction metadata                |

### RpcTransactionInput

| Field              | Type                           | Description                            |
|--------------------|--------------------------------|----------------------------------------|
| `previousOutpoint` | RpcOutpoint                    | Previous transaction outpoint          |
| `signatureScript`  | string                         | Signature script (hex)                 |
| `sequence`         | uint64                         | Sequence number                        |
| `sigOpCount`       | uint32                         | Signature operation count              |
| `verboseData`      | RpcTransactionInputVerboseData | Verbose input metadata                 |

### RpcScriptPublicKey

| Field              | Type   | Description                            |
|--------------------|--------|----------------------------------------|
| `version`          | uint32 | Script public key version              |
| `scriptPublicKey`  | string | Script public key (hex)                |

### RpcTransactionOutput

| Field              | Type                             | Description                              |
|--------------------|----------------------------------|------------------------------------------|
| `amount`           | uint64                           | Amount in sompi                          |
| `scriptPublicKey`  | RpcScriptPublicKey               | Output script public key                 |
| `verboseData`      | RpcTransactionOutputVerboseData  | Verbose output metadata                  |

### RpcOutpoint

| Field           | Type   | Description                        |
|-----------------|--------|------------------------------------|
| `transactionId` | string | Transaction ID (hex)               |
| `index`         | uint32 | Output index within transaction    |

### RpcUtxoEntry

| Field              | Type               | Description                         |
|--------------------|--------------------|-------------------------------------|
| `amount`           | uint64             | Amount in sompi                     |
| `scriptPublicKey`  | RpcScriptPublicKey | Script public key                   |
| `blockDaaScore`    | uint64             | DAA score of the containing block   |
| `isCoinbase`       | bool               | Whether this is a coinbase output   |

### RpcTransactionVerboseData

| Field           | Type   | Description                                    |
|-----------------|--------|------------------------------------------------|
| `transactionId` | string | Transaction ID (hex)                           |
| `hash`          | string | Transaction hash (hex)                         |
| `mass`          | uint64 | Computed transaction mass                      |
| `blockHash`     | string | Hash of the block containing this transaction  |
| `blockTime`     | uint64 | Timestamp of the containing block (Unix ms)    |

### RpcTransactionInputVerboseData

Empty object. No additional input metadata.

### RpcTransactionOutputVerboseData

| Field                    | Type   | Description                                     |
|--------------------------|--------|-------------------------------------------------|
| `scriptPublicKeyType`    | string | Type of the script public key (e.g. `"pubkey"`)|
| `scriptPublicKeyAddress` | string | Derived address from the script public key      |

---

## Error Codes

Eiyaro does not use numeric error codes. All errors are returned as an `RPCError` object with a human-readable `message` field.

**RPCError structure:**

| Field     | Type   | Description             |
|-----------|--------|-------------------------|
| `message` | string | Human-readable error    |

The presence of an `error` field with a non-empty `message` indicates failure. Always check:

```json
{
  "getBlockResponse": {
    "error": {
      "message": "Could not find block with hash abcdef..."
    }
  }
}
```

Common error patterns:

| Error message pattern                        | Meaning                                        |
|----------------------------------------------|------------------------------------------------|
| `"Could not find block ..."`                 | Block hash not found                          |
| `"Could not find transaction ..."`           | Transaction not found                          |
| `"Could not parse ..."`                      | Invalid hash format or malformed input         |
| `"Node is in IBD"`                           | Operation unavailable during initial sync      |
| `"Block is invalid"`                         | Submitted block failed validation              |

---

## Usage Examples

### curl Examples

**Get node info:**
```bash
curl -s -X POST http://localhost:42420 \
  -H "Content-Type: application/json" \
  -d '{"getInfoRequest": {}}'
```

**Get a block:**
```bash
curl -s -X POST http://localhost:42420 \
  -H "Content-Type: application/json" \
  -d '{
    "getBlockRequest": {
      "hash": "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
      "includeTransactions": true
    }
  }'
```

**Get balance:**
```bash
curl -s -X POST http://localhost:42420 \
  -H "Content-Type: application/json" \
  -d '{
    "getBalanceByAddressRequest": {
      "address": "eiyaro:qz..."
    }
  }'
```

**Get multiple balances:**
```bash
curl -s -X POST http://localhost:42420 \
  -H "Content-Type: application/json" \
  -d '{
    "getBalancesByAddressesRequest": {
      "addresses": ["eiyaro:qz...", "eiyaro:qp..."]
    }
  }'
```

**Get UTXOs:**
```bash
curl -s -X POST http://localhost:42420 \
  -H "Content-Type: application/json" \
  -d '{
    "getUtxosByAddressesRequest": {
      "addresses": ["eiyaro:qz..."]
    }
  }'
```

**Get block template:**
```bash
curl -s -X POST http://localhost:42420 \
  -H "Content-Type: application/json" \
  -d '{
    "getBlockTemplateRequest": {
      "payAddress": "eiyaro:qz...",
      "extraData": ""
    }
  }'
```

**Get block count:**
```bash
curl -s -X POST http://localhost:42420 \
  -H "Content-Type: application/json" \
  -d '{"getBlockCountRequest": {}}'
```

**Get DAG info:**
```bash
curl -s -X POST http://localhost:42420 \
  -H "Content-Type: application/json" \
  -d '{"getBlockDagInfoRequest": {}}'
```

**Get coin supply:**
```bash
curl -s -X POST http://localhost:42420 \
  -H "Content-Type: application/json" \
  -d '{"getCoinSupplyRequest": {}}'
```

**Get fee estimate:**
```bash
curl -s -X POST http://localhost:42420 \
  -H "Content-Type: application/json" \
  -d '{"getFeeEstimateRequest": {}}'
```

**Get mempool entries:**
```bash
curl -s -X POST http://localhost:42420 \
  -H "Content-Type: application/json" \
  -d '{
    "getMempoolEntriesRequest": {
      "includeOrphanPool": false,
      "filterTransactionPool": false
    }
  }'
```

**Get mempool entry by tx ID:**
```bash
curl -s -X POST http://localhost:42420 \
  -H "Content-Type: application/json" \
  -d '{
    "getMempoolEntryRequest": {
      "txId": "abcdef1234567890...",
      "includeOrphanPool": false,
      "filterTransactionPool": false
    }
  }'
```

**Get connected peers:**
```bash
curl -s -X POST http://localhost:42420 \
  -H "Content-Type: application/json" \
  -d '{"getConnectedPeerInfoRequest": {}}'
```

**Submit transaction:**
```bash
curl -s -X POST http://localhost:42420 \
  -H "Content-Type: application/json" \
  -d '{
    "submitTransactionRequest": {
      "transaction": {
        "version": 0,
        "inputs": [
          {
            "previousOutpoint": {
              "transactionId": "abcdef1234567890...",
              "index": 0
            },
            "signatureScript": "41...",
            "sequence": "0",
            "sigOpCount": 1
          }
        ],
        "outputs": [
          {
            "amount": "100000000",
            "scriptPublicKey": {
              "version": 0,
              "scriptPublicKey": "76a914..."
            }
          }
        ],
        "lockTime": "0",
        "subnetworkId": "0000000000000000000000000000000000000000000000000000000000000000",
        "gas": "0",
        "payload": ""
      },
      "allowOrphan": false
    }
  }'
```

**Get selected tip hash:**
```bash
curl -s -X POST http://localhost:42420 \
  -H "Content-Type: application/json" \
  -d '{"getSelectedTipHashRequest": {}}'
```

**Get current network:**
```bash
curl -s -X POST http://localhost:42420 \
  -H "Content-Type: application/json" \
  -d '{"getCurrentNetworkRequest": {}}'
```

**Add peer:**
```bash
curl -s -X POST http://localhost:42420 \
  -H "Content-Type: application/json" \
  -d '{
    "addPeerRequest": {
      "address": "192.168.1.100:42420",
      "isPermanent": true
    }
  }'
```

**Ban IP:**
```bash
curl -s -X POST http://localhost:42420 \
  -H "Content-Type: application/json" \
  -d '{"banRequest": {"ip": "192.168.1.100"}}'
```

**Shutdown node:**
```bash
curl -s -X POST http://localhost:42420 \
  -H "Content-Type: application/json" \
  -d '{"shutDownRequest": {}}'
```

### eiyaroctl Examples

The `eiyaroctl` CLI tool provides a more convenient interface:

```bash
eiyaroctl getinfo
eiyaroctl getblock <hash>
eiyaroctl getblocks <lowHash>
eiyaroctl getblockcount
eiyaroctl getblockdaginfo
eiyaroctl getcoinsupply
eiyaroctl getbalance <address>
eiyaroctl getbalances <address1> <address2> ...
eiyaroctl getutxos <address>
eiyaroctl getmempool
eiyaroctl getmempoolentry <txid>
eiyaroctl getpeerinfo
eiyaroctl getpeers
eiyaroctl getselectedtiphash
eiyaroctl getcurrentnetwork
eiyaroctl getfeeestimate
eiyaroctl getblocktemplate <payAddress>
eiyaroctl submitblock <blockJson>
eiyaroctl submittransaction <txJson>
eiyaroctl addpeer <address>
eiyaroctl ban <ip>
eiyaroctl unban <ip>
eiyaroctl resolvefinalityconflict <blockHash>
eiyaroctl shutdown
```

---

## Method Quick Reference

| Category                 | Method Name                                    | Request Key                                    |
|--------------------------|------------------------------------------------|------------------------------------------------|
| Data Retrieval           | GetInfo                                        | `getInfoRequest`                               |
| Data Retrieval           | GetBlockDagInfo                                | `getBlockDagInfoRequest`                       |
| Data Retrieval           | GetBlock                                       | `getBlockRequest`                              |
| Data Retrieval           | GetBlocks                                      | `getBlocksRequest`                             |
| Data Retrieval           | GetBlockCount                                  | `getBlockCountRequest`                         |
| Data Retrieval           | GetHeaders                                     | `getHeadersRequest`                            |
| Data Retrieval           | GetSelectedTipHash                             | `getSelectedTipHashRequest`                    |
| Data Retrieval           | GetCoinSupply                                  | `getCoinSupplyRequest`                         |
| Data Retrieval           | GetBalanceByAddress                            | `getBalanceByAddressRequest`                   |
| Data Retrieval           | GetBalancesByAddresses                         | `getBalancesByAddressesRequest`                |
| Data Retrieval           | GetUtxosByAddresses                            | `getUtxosByAddressesRequest`                   |
| Data Retrieval           | GetMempoolEntries                              | `getMempoolEntriesRequest`                     |
| Data Retrieval           | GetMempoolEntry                                | `getMempoolEntryRequest`                       |
| Data Retrieval           | GetConnectedPeerInfo                           | `getConnectedPeerInfoRequest`                  |
| Data Retrieval           | GetPeerAddresses                               | `getPeerAddressesRequest`                      |
| Data Retrieval           | GetSubnetwork                                  | `getSubnetworkRequest`                         |
| Data Retrieval           | GetBlockTemplate                               | `getBlockTemplateRequest`                      |
| Data Retrieval           | GetFeeEstimate                                 | `getFeeEstimateRequest`                        |
| Data Retrieval           | GetCurrentNetwork                              | `getCurrentNetworkRequest`                     |
| Data Retrieval           | GetBlockByTransactionID                        | `getBlockByTransactionIDRequest`               |
| Data Retrieval           | GetUsableAddresses                             | `getUsableAddressesRequest`                    |
| Data Retrieval           | GetVirtualSelectedParentChainFromBlock         | `getVirtualSelectedParentChainFromBlockRequest`|
| Transaction              | SubmitTransaction                              | `submitTransactionRequest`                     |
| Transaction              | SubmitTransactionReplacement                   | `submitTransactionReplacementRequest`          |
| Block                    | SubmitBlock                                    | `submitBlockRequest`                           |
| Peer                     | AddPeer                                        | `addPeerRequest`                               |
| Peer                     | Ban                                            | `banRequest`                                   |
| Peer                     | Unban                                          | `unbanRequest`                                 |
| Notification             | NotifyBlockAdded                               | `notifyBlockAddedRequest`                      |
| Notification             | NotifyVirtualSelectedParentChainChanged        | `notifyVirtualSelectedParentChainChangedRequest`|
| Notification             | NotifyFinalityConflicts                        | `notifyFinalityConflictsRequest`               |
| Notification             | NotifyUtxosChanged                             | `notifyUtxosChangedRequest`                    |
| Notification             | StopNotifyingUtxosChanged                      | `stopNotifyingUtxosChangedRequest`             |
| Notification             | NotifyNewBlockTemplate                         | `notifyNewBlockTemplateRequest`                |
| System                   | ResolveFinalityConflict                        | `resolveFinalityConflictRequest`               |
| System                   | ShutDown                                       | `shutDownRequest`                              |

---

## FAQ

### How do I enable RPC on my node?

The RPC server is enabled by default when eyarod starts. If it was disabled with `--norpc`, restart the node without that flag. By default the RPC server listens on `localhost` with the network-appropriate port (e.g., `42420` for mainnet). To accept remote connections, configure `rpclisten=0.0.0.0:42420` in your `eiyarod.conf` file.

### Why am I getting "RPC method not found"?

Ensure you are using the correct lowerCamelCase request key in the JSON envelope. Each method wraps its request in a key like `getInfoRequest` (not `GetInfoRequest` or `get_info_request`). If the method is not a registered top-level command, use eiyaroctl's `--json` flag to send the raw JSON directly. Verify the method name against the Method Quick Reference table at the bottom of this document.

### How do I use WebSocket notifications?

Eiyaro does not use a separate WebSocket endpoint. Notifications are pushed over the same HTTP POST connection after you send a `Notify*RequestMessage`. The server begins pushing notification messages asynchronously — your client must continuously read from the connection. Subscribe to `notifyBlockAddedRequest` for new blocks, `notifyUtxosChangedRequest` for UTXO changes (requires `--utxoindex`), or `notifyFinalityConflictsRequest` for finality conflict alerts.

### What's the difference between GetBlock and GetBlocks?

`GetBlock` retrieves a single block by its hash, with an optional flag to include full transaction data. `GetBlocks` returns all block hashes (and optionally block objects) between a `lowHash` and the current virtual tip, traversing the selected parent chain. Use `GetBlock` for looking up a known block and `GetBlocks` for scanning a range of the chain.

### Why do balance queries return errors?

Balance and UTXO queries (`GetBalanceByAddress`, `GetUtxosByAddresses`, `GetBalancesByAddresses`, `GetUsableAddresses`) require the node to have been started with the `--utxoindex` flag. Without it, these calls return an error. Restart eyarod with `--utxoindex` to enable the UTXO index, then wait for the index to finish building before querying.

### How do I interpret the fee estimate response?

The `GetFeeEstimate` response contains three bucket categories: `priorityBucket` (high-priority, fastest confirmation), `normalBuckets`, and `lowBuckets`. Each bucket provides a `feerate` in sompi per gram and an `estimatedSeconds` for inclusion. Choose a feerate based on your urgency — higher feerates in the priority bucket will be included in blocks sooner than those in the low bucket.