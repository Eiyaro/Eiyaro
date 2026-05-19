# Eiyaro P2P Network

## Overview

The Eiyaro peer-to-peer networking layer provides a TCP-based communication protocol with version handshake negotiation, typed message routing, and comprehensive connection lifecycle management. The network supports both full nodes (`SFNodeNetwork`) and partial (subnetwork) nodes, with service flag negotiation occurring during the initial handshake.

The P2P stack is layered as follows:

```
┌──────────────────────────────────────┐
│         Protocol Manager             │
│  Flows: handshake, blockrelay, tx,   │
│  ping, address exchange, rejects     │
├──────────────────────────────────────┤
│            NetAdapter                │
│  (network abstraction layer)         │
├──────────────┬───────────────────────┤
│  P2P Server  │     RPC Server        │
│  (gRPC)      │     (gRPC)            │
├──────────────┴───────────────────────┤
│        ConnectionManager             │
│  (connection lifecycle & quotas)     │
├──────────────────────────────────────┤
│        AddressManager                │
│  (peer discovery, address storage,   │
│   banning, network classification)   │
└──────────────────────────────────────┘
```

---

## Network Architecture

### NetAdapter: Network Abstraction Layer

[netadapter.go](file:///d:/Eiyaro/Eiyaro/infrastructure/network/netadapter/netadapter.go)

The `NetAdapter` is the central abstraction over the network. It manages two independent gRPC servers:

- **P2P Server** — handles peer-to-peer connections from other nodes
- **RPC Server** — handles RPC connections from clients and wallets

Each `NetAdapter` instance is assigned a unique 16-byte ID generated from `crypto/rand` ([id.go](file:///d:/Eiyaro/Eiyaro/infrastructure/network/netadapter/id/id.go)). This ID is exchanged during the version handshake and is used to detect self-connections.

Key responsibilities:
- Exposing `P2PConnect(address)` for outbound peer connections with optional loopback filtering
- Broadcasting messages to all connected P2P peers via `P2PBroadcast()`
- Tracking active connections with read/write mutex protection
- Caching outbound routers for reuse on reconnection (inbound routers are not cached to prevent unbounded memory growth)

### Router: Message Routing Between Peers

[router.go](file:///d:/Eiyaro/Eiyaro/infrastructure/network/netadapter/router/router.go)

Each peer connection is associated with one `Router`. The router maps inbound messages by type to named `Route` channels, and provides a single outgoing route for sending messages to the remote peer.

**Key design features:**
- **Incoming routes**: Type-safe message routing using a `map[MessageCommand]*Route`. Multiple message commands can share a single route (e.g., `CmdInvRelayBlock`, `CmdBlock`, and `CmdBlockLocator` all route to `HandleRelayInvs`).
- **Outgoing route**: A single buffered channel for all messages sent to the peer.
- **Route reuse**: Outbound routers are cached by peer address and reset on reconnection, retaining their buffered channels to avoid reallocation.
- **Capacity**: Default route capacity is 5000 messages (`DefaultMaxMessages`). Transaction relay routes use 10,000 capacity.

**Route mechanics** ([route.go](file:///d:/Eiyaro/Eiyaro/infrastructure/network/netadapter/router/route.go)):
- `Enqueue()` — adds a message to the route; returns `ErrRouteCapacityReached` when full
- `Dequeue()` — reads a message; returns `ErrRouteClosed` if the route has been shut down
- `DequeueWithTimeout()` — blocks with a timeout; returns `ErrTimeout` on expiry
- `Close()` — safely shuts down the route
- `Reset()` — drains queued messages and reopens without reallocating

### ConnectionManager: Connection Lifecycle

[connmanager.go](file:///d:/Eiyaro/Eiyaro/infrastructure/network/connmanager/connmanager.go)

The `ConnectionManager` runs a periodic loop (every 30 seconds) that enforces connection quotas and manages the lifecycle of three connection categories:

| Category | Description |
|---|---|
| **Requested** | User-specified peers (`--addpeer` / `--connect`). Permanent peers are retried indefinitely (30s interval). One-shot peers are removed on failure. |
| **Outgoing** | Automatically discovered outbound peers. Target: `--outpeers` (default **8**). |
| **Incoming** | Peers that connect to us. Max: `--maxinpeers` (default **500**). |

**Connection logic per loop iteration:**

1. **checkRequestedConnections** ([connection_requests.go](file:///d:/Eiyaro/Eiyaro/infrastructure/network/connmanager/connection_requests.go)): Verifies active requested peers are still connected. Retries pending permanent peers with a 30-second backoff. Moves successfully connected pending peers to active.

2. **checkOutgoingConnections** ([outgoing_connections.go](file:///d:/Eiyaro/Eiyaro/infrastructure/network/connmanager/outgoing_connections.go)): Ensures `targetOutgoing` outbound peers are maintained. When below target, obtains random addresses from the `AddressManager` (excluding already-connected peers) and initiates connections. Falls back to DNS seeding when the address pool is insufficient.

3. **checkIncomingConnections** ([incoming_connections.go](file:///d:/Eiyaro/Eiyaro/infrastructure/network/connmanager/incoming_connections.go)): If inbound connections exceed `maxIncoming`, randomly disconnects excess connections.

### AddressManager: Peer Address Discovery and Management

[network.go](file:///d:/Eiyaro/Eiyaro/infrastructure/network/addressmanager/network.go)

The `AddressManager` stores, classifies, and serves peer addresses. It supports:

- **Address discovery** from DNS seeds, gRPC seeds, and peer address exchange (`MsgRequestAddresses` / `MsgAddresses`)
- **Address storage** with scoring and last-seen timestamps
- **Network classification** — categorizes addresses into groups for diversity
- **Address banning** — persists banned addresses with configurable ban duration (default 24 hours, threshold 100)

**Network address classification:**
| Classification | RFC/Description |
|---|---|
| `IsIPv4` | Standard IPv4 |
| `IsLocal` | Loopback + 0.0.0.0/8 |
| `IsRFC1918` | Private: 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16 |
| `IsRFC2544` | Benchmarking: 198.18.0.0/15 |
| `IsRFC3849` | IPv6 documentation: 2001:DB8::/32 |
| `IsRFC3927` | Auto-configuration: 169.254.0.0/16 |
| `IsRFC3964` | 6to4: 2002::/16 |
| `IsRFC4193` | IPv6 unique local: FC00::/7 |
| `IsRFC4380` | Teredo: 2001::/32 |
| `IsRFC4843` | ORCHID: 2001:10::/28 |
| `IsRFC4862` | Link-local: FE80::/64 |
| `IsRFC5737` | Documentation: 192.0.2.0/24, 198.51.100.0/24, 203.0.113.0/24 |
| `IsRFC6052` | Well-known prefix: 64:FF9B::/96 |
| `IsRFC6145` | IPv4-mapped: ::FFFF:0:0:0/96 |
| `IsRFC6598` | Shared address space: 100.64.0.0/10 |
| `IsRoutable` | Publicly reachable (valid, not reserved) |

**Grouping** — The `GroupKey()` method groups addresses for diversity:
- IPv4: /16 prefix
- IPv6: /32 prefix (/36 for Hurricane Electric: 2001:470::/32)
- Special handling for 6to4, Teredo, and IPv4-mapped addresses

**Limits**: Maximum 2500 addresses returned in a single `GetAddresses` response.

---

## P2P Protocol

### Protocol Version

**Current protocol version: 7**

| Constant | Value | Description |
|---|---|---|
| `defaultProtocolVersion` | `7` | Configured protocol version |
| `minAcceptableProtocolVersion` | `7` | Minimum version peers must support |
| `maxAcceptableProtocolVersion` | `7` | Maximum protocol version recognized |

Protocol version is negotiated during the handshake: the lower of the local and advertised remote version is selected. Peers with a version below `minAcceptableProtocolVersion` are rejected.

### Handshake Flow

[handshake.go](file:///d:/Eiyaro/Eiyaro/app/protocol/flows/handshake/handshake.go)

The handshake is a concurrent two-phase exchange with a 60-second timeout per message:

```
Initiator (local)                     Responder (remote)
     │                                      │
     ├──── MsgVersion ────────────────────►│
     │                                      │
     │◄────── MsgVersion ───────────────────┤
     │                                      │
     │◄─────── MsgVerAck ───────────────────┤
     │                                      │
     ├───────── MsgVerAck ────────────────►│
     │                                      │
     ├───────── MsgReady ─────────────────►│
     │◄───────── MsgReady ──────────────────┤
     │                                      │
     ▼                                      ▼
   P2P flows established            P2P flows established
```

**Handshake validation checks:**

1. **Self-connection detection** — Compares remote ID with local ID; rejects self-connections
2. **Network mismatch** — Compares the `Network` field in `MsgVersion`; bans if mismatched (prevents cross-network connections)
3. **Protocol version** — Rejects peers below `minAcceptableProtocolVersion` (7)
4. **Subnetwork compatibility** — Full nodes reject incoming connections from partial nodes on outbound-initiated connections; partial nodes reject peers with incompatible subnetwork IDs
5. **Version enforcement** — When `ForceSameVersion` is enabled, rejects peers with mismatched `htnd` version strings

**After successful handshake:**
- Peer is registered with the protocol manager via `AddToPeers()`
- Remote address is added to the `AddressManager`
- Protocol version 7 flows are registered (see below)
- Handshake routes (`CmdVersion`, `CmdVerAck`, `CmdReady`) are removed
- The `MsgReady` exchange completes initialization
- All registered P2P flows begin executing

### Message Types

#### Version Handshake

| MessageCommand | String | Code | Description |
|---|---|---|---|
| `CmdVersion` | `Version` | 0 | Sent by both peers containing protocol version, services, user agent, timestamp, network, and node ID |
| `CmdVerAck` | `VerAck` | 1 | Acknowledgment of received version message |
| `CmdReady` | `Ready` | 69 | Signals that the peer is ready for P2P message exchange after handshake |

#### Block Relay

| MessageCommand | String | Code | Description |
|---|---|---|---|
| `CmdInvRelayBlock` | `InvRelayBlock` | 44 | Inventory of newly relayed blocks |
| `CmdRequestRelayBlocks` | `RequestRelayBlocks` | 45 | Request for relayed blocks by inventory |
| `CmdBlock` | `Block` | 34 | A full block |
| `CmdBlockHeaders` | `BlockHeaders` | 58 | Block header list |
| `CmdRequestHeaders` | `RequestHeaders` | 35 | Request for block headers |
| `CmdRequestNextHeaders` | `RequestNextHeaders` | 51 | Request for next batch of headers |
| `CmdBlockLocator` | `BlockLocator` | 43 | Block locator list (for sync state) |
| `CmdRequestBlockLocator` | `RequestBlockLocator` | 42 | Request for block locator |
| `CmdBlockWithTrustedData` | `BlockWithTrustedData` | 61 | Block with trusted data (legacy) |
| `CmdBlockWithTrustedDataV4` | `BlockWithTrustedDataV4` | 71 | Block with trusted data v4 |
| `CmdDoneBlocksWithTrustedData` | `DoneBlocksWithTrustedData` | 62 | Signals completion of trusted data blocks |
| `CmdTrustedData` | `TrustedData` | 70 | Trusted data message |
| `CmdRequestPruningPointAndItsAnticone` | `RequestPruningPointAndItsAnticoneHeaders` | 63 | Request pruning point and anticone |
| `CmdRequestNextPruningPointAndItsAnticoneBlocks` | `RequestNextPruningPointAndItsAnticoneBlocks` | 72 | Request next anticone blocks batch |
| `CmdRequestAnticone` | `RequestAnticone` | 75 | Request anticone for a given block |

#### IBD (Initial Block Download)

| MessageCommand | String | Code | Description |
|---|---|---|---|
| `CmdIBDBlock` | `IBDBlock` | 64 | Block sent during IBD |
| `CmdRequestIBDBlocks` | `RequestIBDBlocks` | 65 | Request IBD blocks |
| `CmdDoneHeaders` | `DoneHeaders` | 48 | Signals end of header batch during IBD |
| `CmdIBDBlockLocator` | `IBDBlockLocator` | 55 | IBD block locator |
| `CmdIBDBlockLocatorHighestHash` | `IBDBlockLocatorHighestHash` | 56 | Highest hash in IBD block locator |
| `CmdIBDBlockLocatorHighestHashNotFound` | `IBDBlockLocatorHighestHashNotFound` | 57 | Highest hash not found |
| `CmdIBDChainBlockLocator` | `IBDChainBlockLocator` | 74 | IBD chain block locator |
| `CmdRequestIBDChainBlockLocator` | `RequestIBDChainBlockLocator` | 73 | Request IBD chain block locator |
| `CmdPruningPoints` | `PruningPoints` | 66 | Pruning points list |
| `CmdPruningPointProof` | `PruningPointProof` | 68 | Pruning point proof |
| `CmdRequestPruningPointProof` | `RequestPruningPointProof` | 67 | Request pruning point proof |
| `CmdRequestPruningPointUTXOSet` | `RequestPruningPointUTXOSet` | 52 | Request pruning point UTXO set |
| `CmdPruningPointUTXOSetChunk` | `PruningPointUTXOSetChunk` | 53 | Chunk of pruning point UTXO set |
| `CmdRequestNextPruningPointUTXOSetChunk` | `RequestNextPruningPointUTXOSetChunk` | 59 | Request next UTXO set chunk |
| `CmdDonePruningPointUTXOSetChunks` | `DonePruningPointUTXOSetChunks` | 60 | Signals all UTXO set chunks received |
| `CmdUnexpectedPruningPoint` | `UnexpectedPruningPoint` | 54 | Signals unexpected pruning point |

#### Transactions

| MessageCommand | String | Code | Description |
|---|---|---|---|
| `CmdInvTransaction` | `InvTransaction` | 46 | Inventory of new transactions |
| `CmdTx` | `Tx` | 36 | A full transaction |
| `CmdRequestTransactions` | `RequestTransactions` | 47 | Request transactions by ID |
| `CmdTransactionNotFound` | `TransactionNotFound` | 49 | Transaction not found in mempool |

#### Address Exchange

| MessageCommand | String | Code | Description |
|---|---|---|---|
| `CmdRequestAddresses` | `RequestAddresses` | 32 | Request peer addresses |
| `CmdAddresses` | `Addresses` | 33 | Peer address list (max 2500 per response) |

#### Ping/Pong

| MessageCommand | String | Code | Description |
|---|---|---|---|
| `CmdPing` | `Ping` | 38 | Ping with nonce |
| `CmdPong` | `Pong` | 41 | Pong echoing nonce |

#### Reject

| MessageCommand | String | Code | Description |
|---|---|---|---|
| `CmdReject` | `Reject` | 50 | Rejection of a message with reason |

### Service Flags

Service flags are 64-bit bitfields advertised in `MsgVersion` to indicate a node's capabilities.

| Service Flag | Value | Description |
|---|---|---|
| `SFNodeNetwork` | `1 << 0` = `1` | Full node — stores and serves the full block DAG |
| `SFNodeGetUTXO` | `1 << 1` = `2` | Supports `getutxos` / `utxos` commands (BIP0064) |
| `SFNodeBloom` | `1 << 2` = `4` | Supports bloom filtering |
| `SFNodeXthin` | `1 << 3` = `8` | Supports xthin (compact) blocks |
| `SFNodeBit5` | `1 << 4` = `16` | Bit 5 service flag |
| `SFNodeCF` | `1 << 5` = `32` | Supports committed filters (CFs) |

**Default services**: `SFNodeNetwork | SFNodeBloom | SFNodeCF` (value = 37)

### Network Magic

The `EiyaroNet` value is embedded in each message header to prevent cross-network replay:

| Network | Magic Value |
|---|---|
| `Mainnet` | `0x3ddcf71d` |
| `Testnet` | `0xddb8af8f` |
| `Simnet` | `0x374dcf1c` |
| `Devnet` | `0x732d87e1` |

**Max message payload**: 32 MB (`MaxMessagePayload = 1024 * 1024 * 32`)

---

## DNS Seeding

### How Initial Peer Discovery Works

[dnsseed/seed.go](file:///d:/Eiyaro/Eiyaro/infrastructure/network/dnsseed/seed.go)

When the node has fewer than `TargetOutboundPeers` (default 8) outbound connections, it seeds from DNS and gRPC sources to populate the `AddressManager`:

1. **DNS seeding** (`SeedFromDNS`): Queries DNS seed servers for peer addresses
   - Performs a DNS lookup against the seed hostname
   - The hostname format supports subnetwork filtering:
     - All subnetworks: direct host lookup
     - Specific subnetwork: `n<subnetworkID>.<seedhost>`
     - No subnetwork (full node): `n.<seedhost>` (prefix char: `'n'` = `0x6E`)
   - Parses returned IP addresses and wraps them as `NetAddress` with the default port
   - Timestamps are randomly set between 3 and 7 days ago

2. **gRPC seeding** (`SeedFromGRPC`): Queries gRPC seed servers
   - Connects via gRPC to configured seed hosts
   - Sends `GetPeersListRequest` with optional `SubnetworkID` filter
   - Receives and processes the peer address list

**Seeding is asynchronous** — the results populate the `AddressManager` and become available for the next connection loop iteration (30 seconds).

---

## Connection Management

### Connection Limits

| Parameter | Default | CLI Flag | Description |
|---|---|---|---|
| Target outbound | 8 | `--outpeers` | Number of outbound peers to maintain |
| Max inbound | 500 | `--maxinpeers` | Maximum inbound peer connections |
| RPC max clients | 500 | (config) | Maximum simultaneous RPC clients |
| Total peers | 508 (recommended) | — | outbound + inbound |

When `--connect` is specified, `TargetOutboundPeers` is forced to 1 and DNS seeding is disabled.

### Connection Lifecycle

```
 Pending Requested ──connect──► Active Requested
                                     │
 DNS/AddrManager ──connect──► Active Outgoing
                                     │
 Remote Peer ────connect─────► Active Incoming
                                     │
                            ┌────────┴─────────┐
                            ▼                   ▼
                     Handshake OK         Handshake Failed
                            │                   │
                     P2P Flows Start       Disconnect
                            │              (ban if protocol error)
                     Message Exchange
                            │
                    Disconnect / Error
                            │
                  Remove from active set
             (retry if permanent requested)
```

### Ban Scoring

Bans are managed by the `AddressManager`:

- **Ban duration**: 24 hours (configurable via `--banduration`)
- **Ban threshold**: 100 (configurable via `--banthreshold`)
- **Permanent peers cannot be banned** (`ErrCannotBanPermanent` is returned)
- Bans can be applied by:
  - `ConnectionManager.Ban()` — bans a specific `NetConnection`
  - `ConnectionManager.BanByIP()` — bans an IP address, disconnecting all peers on that IP
- When a protocol error with `ShouldBan = true` occurs, the peer is banned and a `MsgReject` with the error reason is sent before disconnection

---

## Address Management

### Address Discovery

Addresses are discovered through three mechanisms:

1. **DNS seeding** — Initial bootstrap from DNS seed servers
2. **gRPC seeding** — Bootstrap from gRPC seed servers
3. **Peer address exchange** — The `SendAddresses` / `ReceiveAddresses` flows exchange addresses between connected peers. When a peer requests addresses (`MsgRequestAddresses`), up to 2500 random addresses are returned.

### Address Storage

The `AddressManager` maintains an internal store of known peer addresses with:
- IP and port
- Last seen timestamp
- Connection attempt count and success/failure scoring
- Network group classification for address diversity

### Network Classification

Addresses are classified using the extensive RFC-based checks documented in [network.go](file:///d:/Eiyaro/Eiyaro/infrastructure/network/addressmanager/network.go). This enables:
- Filtering out non-routable and reserved addresses
- Grouping addresses to prevent address selection bias (e.g., not selecting all peers from the same /16 subnet)
- Proper handling of IPv4, IPv6, IPv4-mapped, 6to4, and Teredo addresses

### Address Banning

Banned addresses are stored persistently. When a banned peer attempts to connect, the connection is rejected during router initialization with a log message and immediate disconnect.

---

## Initial Block Download (IBD)

### How IBD Works

IBD is the process by which a new or out-of-sync node downloads the block DAG from a single "IBD peer":

1. **Peer selection**: When the node's consensus state indicates it is not synced, one peer is designated as the IBD peer (stored in `FlowContext.ibdPeer` with mutex protection).

2. **Block locator exchange**: The node sends block locator messages to determine which blocks it needs:
   - `MsgRequestBlockLocator` / `MsgBlockLocator` — standard sync state exchange
   - `MsgIBDBlockLocator` — IBD-specific locator
   - `MsgIBDChainBlockLocator` — chain block locator for IBD
   - `MsgRequestIBDChainBlockLocator` — request for IBD chain block locator

3. **Header synchronization**: Headers are requested and received in batches:
   - `MsgRequestHeaders` / `MsgRequestNextHeaders` — request headers
   - `MsgBlockHeaders` — received header batch
   - `MsgDoneHeaders` — signals end of header batch

4. **Block download**: After headers are synced, blocks are downloaded:
   - `MsgRequestIBDBlocks` — request blocks
   - `MsgIBDBlock` — received IBD block
   - `MsgBlockWithTrustedDataV4` — block with trusted data

5. **Pruning point synchronization**: The pruning point and its UTXO set are requested:
   - `MsgRequestPruningPointUTXOSet` — request UTXO set
   - `MsgPruningPointUTXOSetChunk` — received UTXO set chunk
   - `MsgRequestNextPruningPointUTXOSetChunk` — request next chunk
   - `MsgDonePruningPointUTXOSetChunks` — all chunks received
   - `MsgPruningPoints` / `MsgPruningPointProof` — pruning point data

### IBD Flow Registration

All IBD messages route to the single `HandleIBD` flow registered in [v7/register.go](file:///d:/Eiyaro/Eiyaro/app/protocol/flows/v7/register.go). This flow handles 14 message types covering the entire IBD lifecycle.

### Sync State Management

The `FlowContext` exposes `IsNearlySynced()` which queries the consensus engine to determine current sync progress. The `ibdPeer` field tracks which peer is currently serving IBD data, protected by a read-write mutex.

---

## Protocol Flows (Version 7)

After the handshake completes, the following flows are registered for protocol version 7:

| Flow Name | Type | Messages Handled | Purpose |
|---|---|---|---|
| `SendAddresses` | Persistent | `CmdRequestAddresses` | Responds to address requests |
| `ReceiveAddresses` | One-time | `CmdAddresses` | Processes received addresses |
| `SendVirtualSelectedParentInv` | One-time | (none) | Sends virtual selected parent inventory |
| `HandleRelayInvs` | Persistent | `CmdInvRelayBlock`, `CmdBlock`, `CmdBlockLocator` | Processes block relay inventories |
| `HandleIBD` | Persistent | `CmdDoneHeaders`, `CmdBlockHeaders`, `CmdIBDBlock` (14 types) | Manages IBD process |
| `HandleRelayBlockRequests` | Persistent | `CmdRequestRelayBlocks` | Serves relay block requests |
| `HandleRequestBlockLocator` | Persistent | `CmdRequestBlockLocator` | Serves block locator requests |
| `HandleRequestHeaders` | Persistent | `CmdRequestHeaders`, `CmdRequestNextHeaders` | Serves header requests |
| `HandleIBDBlockRequests` | Persistent | `CmdRequestIBDBlocks` | Serves IBD block requests |
| `HandleRequestPruningPointUTXOSet` | Persistent | `CmdRequestPruningPointUTXOSet`, `CmdRequestNextPruningPointUTXOSetChunk` | Serves pruning point UTXO set |
| `HandlePruningPointAndItsAnticoneRequests` | Persistent | `CmdRequestPruningPointAndItsAnticone`, `CmdRequestNextPruningPointAndItsAnticoneBlocks` | Serves pruning point and anticone |
| `HandleIBDBlockLocator` | Persistent | `CmdIBDBlockLocator` | Processes IBD block locator |
| `HandleRequestIBDChainBlockLocator` | Persistent | `CmdRequestIBDChainBlockLocator` | Serves IBD chain block locator |
| `HandleRequestAnticone` | Persistent | `CmdRequestAnticone` | Serves anticone requests |
| `HandlePruningPointProofRequests` | Persistent | `CmdRequestPruningPointProof` | Serves pruning point proof requests |
| `ReceivePings` | Persistent | `CmdPing` | Responds to pings |
| `SendPings` | Persistent | `CmdPong` | Verifies ping responses and tracks latency |
| `HandleRelayedTransactions` | Persistent | `CmdInvTransaction`, `CmdTx`, `CmdTransactionNotFound` | Processes relayed transactions |
| `HandleRequestTransactions` | Persistent | `CmdRequestTransactions` | Serves transaction requests |
| `HandleRejects` | Persistent | `CmdReject` | Processes rejection messages |

---

## RPC Client

[rpcclient.go](file:///d:/Eiyaro/Eiyaro/infrastructure/network/rpcclient/rpcclient.go)

The `RPCClient` provides a gRPC-based interface for wallet and client communication:

- **Connection**: Establishes a gRPC connection to the specified address
- **Version check**: On initial connection, compares local version (`1.0.0`) with the remote server version; warns on mismatch
- **Timeout**: Default RPC timeout is 10 minutes; initial version check uses 5 seconds
- **Reconnection**: Automatically reconnects on disconnect with a 10-second retry interval
- **Router**: Each RPC client builds a full router with individual routes for every RPC message command type (80+ commands)

---

## Usage Examples

### Configuring Persistent Peers

Add a peer that the node will always maintain a connection with:

```ini
# eiyarod.conf
addpeer=192.168.1.100:16111
addpeer=10.0.0.50:16111
```

Or via CLI:

```bash
eiyarod --addpeer=192.168.1.100:16111
```

Persistent peers are retried every 30 seconds if disconnected.

### Connect-Only Mode

Connect exclusively to specific peers (no DNS seeding, no auto-discovered outbound peers):

```bash
eiyarod --connect=trusted-node.example.com:16111
```

This forces `TargetOutboundPeers = 1` and disables DNS seeding.

### Setting Up DNS Seeds

Specify a custom DNS seed:

```bash
eiyarod --dnsseed=seed.example.com
```

Specify a custom gRPC seed:

```bash
eiyarod --grpcseed=grpc-seed.example.com:50051
```

Disable DNS seeding entirely:

```bash
eiyarod --nodnsseed
```

### Connection Troubleshooting

**Adjust connection limits:**

```bash
# Increase outbound peer target
eiyarod --outpeers=16

# Limit inbound connections
eiyarod --maxinpeers=100
```

**Disable banning** (useful for development):

```bash
eiyarod --nobanning
```

**Force same version** (reject peers with different `htnd` versions):

```bash
eiyarod --force-same-version
```

**Disallow loopback P2P connections:**

Handled automatically via NetAdapter — loopback (`127.0.0.1`, `::1`, `localhost`) P2P connections are rejected when configured.

**Ban a peer via RPC:**

```bash
eiyaroctl ban <IP or address>
```

**View connected peers:**

```bash
eiyaroctl getConnectedPeerInfo
```

---

## Key Constants Reference

| Constant | Value | Location |
|---|---|---|
| Protocol Version | `7` | [config.go](file:///d:/Eiyaro/Eiyaro/infrastructure/config/config.go) |
| Min Acceptable Version | `7` | [receiveversion.go](file:///d:/Eiyaro/Eiyaro/app/protocol/flows/handshake/receiveversion.go) |
| Max Acceptable Version | `7` | [receiveversion.go](file:///d:/Eiyaro/Eiyaro/app/protocol/flows/handshake/receiveversion.go) |
| Target Outbound Peers | `8` | [config.go](file:///d:/Eiyaro/Eiyaro/infrastructure/config/config.go) |
| Max Inbound Peers | `500` | [config.go](file:///d:/Eiyaro/Eiyaro/infrastructure/config/config.go) |
| Connection Loop Interval | `30s` | [connmanager.go](file:///d:/Eiyaro/Eiyaro/infrastructure/network/connmanager/connmanager.go) |
| Handshake Timeout | `60s` | [sendversion.go](file:///d:/Eiyaro/Eiyaro/app/protocol/flows/handshake/sendversion.go) / [receiveversion.go](file:///d:/Eiyaro/Eiyaro/app/protocol/flows/handshake/receiveversion.go) |
| Reconnect Retry | `30s` | [connection_requests.go](file:///d:/Eiyaro/Eiyaro/infrastructure/network/connmanager/connection_requests.go) |
| RPC Reconnect Retry | `10s` | [rpcclient.go](file:///d:/Eiyaro/Eiyaro/infrastructure/network/rpcclient/rpcclient.go) |
| Route Capacity (default) | `5000` | [route.go](file:///d:/Eiyaro/Eiyaro/infrastructure/network/netadapter/router/route.go) |
| Route Capacity (tx relay) | `10000` | [v7/register.go](file:///d:/Eiyaro/Eiyaro/app/protocol/flows/v7/register.go) |
| Max Message Payload | `32 MB` | [message.go](file:///d:/Eiyaro/Eiyaro/app/appmessage/message.go) |
| Max Addresses Returned | `2500` | [network.go](file:///d:/Eiyaro/Eiyaro/infrastructure/network/addressmanager/network.go) |
| Default Ban Duration | `24h` | [config.go](file:///d:/Eiyaro/Eiyaro/infrastructure/config/config.go) |
| Default Ban Threshold | `100` | [config.go](file:///d:/Eiyaro/Eiyaro/infrastructure/config/config.go) |
| Node ID Length | `16 bytes` | [id.go](file:///d:/Eiyaro/Eiyaro/infrastructure/network/netadapter/id/id.go) |
| Default Services | `SFNodeNetwork \| SFNodeBloom \| SFNodeCF` (`37`) | [protocol.go](file:///d:/Eiyaro/Eiyaro/app/appmessage/protocol.go) |
| User Agent | `htnd:1.0.0` | [sendversion.go](file:///d:/Eiyaro/Eiyaro/app/protocol/flows/handshake/sendversion.go) |
| Mainnet Magic | `0x3ddcf71d` | [protocol.go](file:///d:/Eiyaro/Eiyaro/app/appmessage/protocol.go) |
| Testnet Magic | `0xddb8af8f` | [protocol.go](file:///d:/Eiyaro/Eiyaro/app/appmessage/protocol.go) |

---

## Notes & Caveats

### Sync Race Conditions

During the initial connection loop, the node may briefly have zero outbound peers while the DNS seeding results are being fetched asynchronously. This gap (up to one connection loop interval, 30 seconds) is expected. The node will establish connections in the next loop iteration once the AddressManager has been populated.

### Connection Loop Timing

The `ConnectionManager` runs every 30 seconds. This means newly discovered peers from address exchange or DNS seeding in iteration N will only be dialed in iteration N+1. For nodes behind restrictive firewalls, this can make the initial peer discovery feel slow — allow at least 2–3 minutes for the first outbound connections to stabilize.

### Ban Persistence

Banned addresses are stored persistently by the `AddressManager`. If you disable banning (`--nobanning`), previously banned addresses remain in the ban store and are not automatically cleared. To reset all bans, remove the address manager's persistent storage (within the application data directory). Permanent peers (`--addpeer`) are exempt from banning and will always be retried regardless of ban score.

### Network Mismatch Detection

If a peer connects with a mismatched `Network` field in `MsgVersion` (e.g., a testnet peer connecting to a mainnet node), the connection is rejected and the peer is banned immediately. This prevents cross-network contamination but means accidentally connecting to the wrong network will get you banned from honest nodes.

### Route Capacity

Default route capacity is 5,000 messages. When a route is full, `Enqueue()` returns `ErrRouteCapacityReached` and the message is dropped. Transaction relay routes use a higher 10,000 capacity. Under extreme conditions (e.g., mass transaction spam), routes may silently drop messages — this is by design to prevent memory exhaustion.

---

## API Reference

### Protocol Message Types Summary

| Category | Commands |
|---|---|
| **Handshake** | `Version (0)`, `VerAck (1)`, `Ready (69)` |
| **Block Relay** | `InvRelayBlock (44)`, `RequestRelayBlocks (45)`, `Block (34)`, `BlockHeaders (58)`, `RequestHeaders (35)`, `RequestNextHeaders (51)`, `BlockLocator (43)`, `RequestBlockLocator (42)`, `BlockWithTrustedData (61)`, `BlockWithTrustedDataV4 (71)`, `DoneBlocksWithTrustedData (62)`, `TrustedData (70)`, `RequestPruningPointAndItsAnticoneHeaders (63)`, `RequestNextPruningPointAndItsAnticoneBlocks (72)`, `RequestAnticone (75)` |
| **IBD** | `IBDBlock (64)`, `RequestIBDBlocks (65)`, `DoneHeaders (48)`, `IBDBlockLocator (55)`, `IBDBlockLocatorHighestHash (56)`, `IBDBlockLocatorHighestHashNotFound (57)`, `IBDChainBlockLocator (74)`, `RequestIBDChainBlockLocator (73)`, `PruningPoints (66)`, `PruningPointProof (68)`, `RequestPruningPointProof (67)`, `RequestPruningPointUTXOSet (52)`, `PruningPointUTXOSetChunk (53)`, `RequestNextPruningPointUTXOSetChunk (59)`, `DonePruningPointUTXOSetChunks (60)`, `UnexpectedPruningPoint (54)` |
| **Transactions** | `InvTransaction (46)`, `Tx (36)`, `RequestTransactions (47)`, `TransactionNotFound (49)` |
| **Address Exchange** | `RequestAddresses (32)`, `Addresses (33)` |
| **Ping/Pong** | `Ping (38)`, `Pong (41)` |
| **Reject** | `Reject (50)` |

### Message Routing Architecture

```
Peer Connection
    │
    ▼
  Router
    ├── Incoming Routes (map[MessageCommand]*Route)
    │     ├── CmdInvRelayBlock → HandleRelayInvs
    │     ├── CmdBlock → HandleRelayInvs
    │     ├── CmdBlockLocator → HandleRelayInvs
    │     ├── CmdPing → ReceivePings
    │     ├── CmdPong → SendPings
    │     ├── CmdTx → HandleRelayedTransactions
    │     ├── CmdInvTransaction → HandleRelayedTransactions
    │     ├── CmdTransactionNotFound → HandleRelayedTransactions
    │     ├── CmdReject → HandleRejects
    │     ├── CmdAddresses → ReceiveAddresses
    │     ├── CmdRequestAddresses → SendAddresses
    │     ├── CmdBlockHeaders → HandleIBD
    │     ├── CmdIBDBlock → HandleIBD
    │     ├── CmdDoneHeaders → HandleIBD
    │     ├── CmdRequestRelayBlocks → HandleRelayBlockRequests
    │     ├── CmdRequestHeaders → HandleRequestHeaders
    │     ├── CmdRequestNextHeaders → HandleRequestHeaders
    │     ├── CmdRequestBlockLocator → HandleRequestBlockLocator
    │     ├── CmdRequestIBDBlocks → HandleIBDBlockRequests
    │     ├── CmdRequestPruningPointUTXOSet → HandleRequestPruningPointUTXOSet
    │     ├── CmdRequestNextPruningPointUTXOSetChunk → HandleRequestPruningPointUTXOSet
    │     ├── CmdRequestPruningPointAndItsAnticone → HandlePruningPointAndItsAnticoneRequests
    │     ├── CmdRequestNextPruningPointAndItsAnticoneBlocks → HandlePruningPointAndItsAnticoneRequests
    │     ├── CmdIBDBlockLocator → HandleIBDBlockLocator
    │     ├── CmdRequestIBDChainBlockLocator → HandleRequestIBDChainBlockLocator
    │     ├── CmdRequestAnticone → HandleRequestAnticone
    │     ├── CmdRequestPruningPointProof → HandlePruningPointProofRequests
    │     └── CmdRequestTransactions → HandleRequestTransactions
    │
    └── Outgoing Route (single buffered channel, capacity 5000/10000)
```

---

## FAQ

### Why can't my node connect to any peers?

The most common cause is a firewall blocking outbound connections on the P2P port (default 42421 for mainnet). Also check that you have not specified `--connect` with unreachable addresses, which disables DNS seeding. If the AddressManager has an empty or stale address pool, try specifying `--dnsseed=seed.example.com` to bootstrap discovery.

### How do I connect to a specific node?

Use `--addpeer=IP:PORT` to add a persistent peer that will be retried every 30 seconds, or `--connect=IP:PORT` to connect exclusively to that peer (disables all other peer discovery). For mainnet the default port is 42421; for testnet-10 it is 42423.

### What port should I open in my firewall?

Open the P2P port for both TCP inbound and outbound traffic. The default mainnet P2P port is **42421**. If you want to accept incoming connections and contribute to the network, ensure this port is reachable from the internet and set `--externalip=YOUR_PUBLIC_IP` if behind NAT.

### How do I check if my node is properly connected?

Use `eiyaroctl getConnectedPeerInfo` to list all connected peers, their addresses, and connection directions (inbound/outbound). A healthy mainnet node should have approximately 8 outbound peers and a variable number of inbound peers. Also check `eiyaroctl getInfo` to verify the sync status and block height.

### What does "protocol version mismatch" mean?

Eiyaro nodes negotiate protocol version during the handshake. Both peers must agree on version 7 (the current protocol version). If your node is running an older or newer incompatible version, connections will be rejected. Use `--force-same-version` to reject peers with different htnd version strings.