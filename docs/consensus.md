# Eiyaro Consensus Engine

## Overview

The consensus engine is the core of Eiyaro, implementing the **GhostDAG** (GHOST Directed Acyclic Graph) protocol — a generalization of Nakamoto consensus for DAG-based blockchains. It handles block validation, ordering, finality determination, UTXO management, pruning, and database management.

Eiyaro's consensus is defined by the interface at [consensus.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/model/externalapi/consensus.go#L4-L63). The concrete implementation lives in [consensus.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/consensus.go#L23-L71) and is constructed via the `Factory` in [factory.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/factory.go#L97-L110).

### Key Concepts

- **Block DAG**: Unlike Bitcoin's linear chain, Eiyaro blocks can reference multiple parent blocks, forming a Directed Acyclic Graph
- **GhostDAG**: The protocol that orders blocks in the DAG, classifying each block's merge set into **blue** (honest/confirmed) and **red** (orphaned) sets
- **Blue Score**: A block's accumulated chain work — the primary ordering metric, equivalent to chain height in Bitcoin
- **Virtual Block**: A synthetic "tip" block whose parents are the current DAG tips. It represents the node's current view of consensus state
- **Selected Parent Chain**: The heaviest-weight path through the DAG, analogous to Bitcoin's longest chain
- **Pruning Point**: The oldest block for which full data is retained; blocks older than this are pruned to headers-only

---

## Architecture

The consensus engine uses a modular design with specialized **process managers** coordinated through the main `consensus` struct:

```
Consensus Interface
├── BlockValidator       — validates blocks in isolation and in context
├── BlockProcessor       — processes and inserts validated blocks
├── GhostDAGManager      — computes blue/red sets, merge sets, blue score
├── DAGTopologyManager   — manages DAG connectivity (parents, children, ancestry)
├── DAGTraversalManager  — traverses the DAG (iterators, heaps, anticone computation)
├── DifficultyManager    — adjusts mining difficulty (DAA)
├── CoinbaseManager      — handles coinbase transactions and block subsidies
├── ConsensusStateManager — resolves virtual state, manages UTXO diffs
├── FinalityManager      — determines block finality points
├── PruningManager       — prunes old block data, advances pruning point
├── PruningProofManager  — generates and validates pruning proofs for IBD
├── SyncManager          — manages sync state, block locators, anticone queries
├── ReachabilityManager  — queries block reachability in sub-linear time
├── BlockBuilder         — constructs new blocks from the current state
├── PastMedianTimeManager — computes the past median time for timestamp validation
├── MergeDepthManager    — manages merge depth constraints
├── HeadersSelectedTipManager — tracks the header-only selected tip
├── TransactionValidator — validates transactions in isolation and in context
└── BlockParentBuilder   — resolves and validates block parents
```

### Data Stores

All persistent state is managed through dedicated **data stores** (key-value wrappers over LevelDB), each with configurable LRU caches:

| Store | Purpose |
|-------|---------|
| `BlockStore` | Full blocks (header + transactions) |
| `BlockHeaderStore` | Block headers (retained even after pruning) |
| `BlockStatusStore` | Block validation status (`Invalid`, `UTXOValid`, `HeaderOnly`, etc.) |
| `BlockRelationStore` | Parent/child relationships per block level |
| `GHOSTDAGDataStore` | GhostDAG data per block (blue score, blue work, selected parent, merge sets, anticone sizes) |
| `ReachabilityDataStore` | Reachability tree data for sub-linear ancestor queries |
| `UTXODiffStore` | UTXO diffs (additions/removals per block) |
| `MultisetStore` | Multiset hashes for UTXO commitment verification |
| `AcceptanceDataStore` | Per-block acceptance data (which transactions were accepted) |
| `ConsensusStateStore` | Virtual UTXO set, tips, and global consensus state |
| `PruningStore` | Pruning point, pruning point UTXO set, pruning candidates |
| `DAABlocksStore` | DAA scores and DAA-added block sets |
| `FinalityStore` | Finality point per block |
| `HeadersSelectedTipStore` | The current header-only selected tip |
| `HeadersSelectedChainStore` | Selected chain data for header-only blocks |
| `WindowHeapSliceStore` | Pre-computed heap slices for DAA window queries |

---

## GhostDAG Algorithm

GhostDAG is the core ordering protocol. It generalizes Nakamoto's "longest chain wins" rule to a DAG by computing, for each new block, which of the blocks in its past belong to the "honest" (blue) set and which are orphaned (red).

### Algorithm Implementation

The primary implementation is in `ghostdagmanager` at [ghostdag.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/processes/ghostdagmanager/ghostdag.go). An alternative implementation exists at [ghostdag2/ghostdagimpl.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/processes/ghostdag2/ghostdagimpl.go).

#### Phase 1: Selected Parent Selection

```go
// Find the parent with the highest blue work
func (gm *ghostdagManager) findSelectedParent(stagingArea, blockParents) (*externalapi.DomainHash, error)
```

The **selected parent** is the parent with the highest `BlueWork`. In case of a tie, the parent with the larger hash (lexicographically) wins. This ensures a deterministic tie-breaking rule.

#### Phase 2: Merge Set Computation

The **merge set** is the set of all blocks in the past of the new block that are NOT in the past of the selected parent, plus the selected parent itself. This is computed by a BFS walk from all parents, stopping when reaching blocks that are ancestors of the selected parent.

```
mergeSetWithoutSelectedParent = BFS from all parents, excluding ancestors of selectedParent
```

#### Phase 3: Blue/Red Classification

For each block in the merge set (sorted by descending blue work), the algorithm checks whether adding it as **blue** would violate the **K-cluster** condition:

1. **Maximum anticone size** (Condition 1): For the candidate block, check that its anticone size with the current blue set does not exceed **K**. If `|anticone-of-candidate ∩ blue-set| > K`, the block is **red**.

2. **K-cluster integrity** (Condition 2): For every existing blue block that is in the anticone of the candidate, verify that adding the candidate does not push that blue block's anticone size past **K**. If any blue block's blue anticone would exceed **K**, the candidate is **red**.

Both conditions are checked efficiently using `BluesAnticoneSizes`, a map that tracks, for each blue block, how many other blue blocks are in its anticone. This map is maintained incrementally and inherited down the selected parent chain.

#### Phase 4: Blue Score and Blue Work

```go
newBlockData.blueScore = selectedParentGHOSTDAGData.BlueScore() + len(newBlockData.mergeSetBlues)
newBlockData.blueWork = selectedParentGHOSTDAGData.BlueWork() + sum(CalcWork(header.Bits()) for each blue in mergeSetBlues)
```

- **Blue Score** = selected parent's blue score + number of merge set blues (including selected parent itself)
- **Blue Work** = selected parent's blue work + accumulated proof-of-work of each blue block in the merge set

### K Parameter

**K** (default: `18` for mainnet) is the maximum allowed size of the anticone intersection between any blue block and the blue set. It is defined in [consensus_defaults.go](file:///d:/Eiyaro/Eiyaro/domain/dagconfig/consensus_defaults.go#L44):

```go
defaultGHOSTDAGK = 18
```

K is version-dependent, allowing different values per block version (accessed via `gm.k[constants.GetBlockVersion()-1]`). Higher K means more blocks can be blue, increasing throughput but also increasing the bound on network round-trip time.

### Block Ordering

The DAG is ordered by:
1. **Topological sort** (ancestors before descendants)
2. Within each topological layer, **blue work** (descending) — blues take priority over reds
3. For equal blue work, **hash comparison** (deterministic tie-breaking)

This ordering ensures all nodes converge on the same transaction ordering without coordination.

### Selected Parent Chain

The selected parent chain is the path from genesis to the virtual block following the `SelectedParent` pointer at each step. It is the "heaviest" chain through the DAG, analogous to Bitcoin's longest chain. Blocks on this chain are the most likely to be confirmed.

---

## Block Structure

### DomainBlock

```go
// From block.go
type DomainBlock struct {
    Header       BlockHeader
    Transactions []*DomainTransaction
    PoWHash      string
}
```

### BlockHeader (Interface)

```go
type BaseBlockHeader interface {
    Version() uint16
    Parents() []BlockLevelParents           // Multi-level parent references
    DirectParents() BlockLevelParents       // Level-0 (direct) parents
    HashMerkleRoot() *DomainHash
    AcceptedIDMerkleRoot() *DomainHash
    UTXOCommitment() *DomainHash
    TimeInMilliseconds() int64
    Bits() uint32                          // Compact difficulty target
    Nonce() uint64
    DAAScore() uint64                      // DAA (Difficulty Adjustment Algorithm) score
    BlueScore() uint64                     // GhostDAG blue score
    BlueWork() *big.Int                    // Accumulated blue work
    PruningPoint() *DomainHash             // The pruning point as of this block
    BlockLevel(maxBlockLevel int) int      // Block level for multi-level DAG
    Equal(other BaseBlockHeader) bool
}
```

### Block Multi-Level Structure

Eiyaro uses a **multi-level DAG** where blocks can reference parents at different block levels. Level 0 is the direct parent level. Higher levels allow referencing ancestors further back. This is managed by the `ParentsManager` and `BlockParentBuilder`.

---

## Block Validation Pipeline

The validation pipeline is implemented in the `blockvalidator` package and called by `blockprocessor`. Each validation step can return a `ruleerrors` error that classifies the specific violation.

### 1. Header Validation in Isolation

Performed by `validateHeaderInIsolation` at [block_header_in_isolation.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/processes/blockvalidator/block_header_in_isolation.go):

- **Block version check**: Version must match the expected version for the block's DAA score
- **Timestamp in isolation**: Timestamp must not be too far in the future (within `TimestampDeviationTolerance`)
- **Parent count limits**: Number of direct parents must be within `MaxBlockParents` for the block version

### 2. Body Validation in Isolation

Performed by `validateBodyInIsolation` at [block_body_in_isolation.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/processes/blockvalidator/block_body_in_isolation.go):

- Transactions are individually validated in isolation via `TransactionValidator.ValidateTransactionInIsolation`
- Block mass check: total mass must not exceed `MaxBlockMass`
- Merge set size check: merge set size must not exceed `MergeSetSizeLimit`

### 3. Pruning Point Validation

Performed at [header_pruning_point.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/processes/blockvalidator/header_pruning_point.go):

- Validates that the block's pruning point header field is correct
- Checks that the block is at sufficient depth from the pruning point

### 4. Header Validation in Context

Performed by `validateHeaderInContext` at [block_header_in_context.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/processes/blockvalidator/block_header_in_context.go):

- **Proof-of-work validation**: Block hash must satisfy the required difficulty target (unless `skipPoW` is set)
- **Timestamp in context**: Timestamp must be greater than the past median time
- **Difficulty check**: Block bits must match the required difficulty from `DifficultyManager.RequiredDifficulty`
- **Parent connectivity**: All parents must exist and not introduce circular dependencies
- **Merge depth check**: Block must satisfy merge depth constraints

### 5. Body Validation in Context

Performed by `validateBodyInContext` at [block_body_in_context.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/processes/blockvalidator/block_body_in_context.go):

- **Coinbase transaction validation**: The first transaction must be a valid coinbase
- **Transaction validation in context**: Each transaction must be valid against the UTXO set (no double-spends, valid signatures, etc.)
- **UTXO commitment verification**: The computed UTXO set multiset hash must match `header.UTXOCommitment()`

### 6. Pruning Violation PoW Check

Performed at [pruning_violation_proof_of_work_and_difficulty.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/processes/blockvalidator/pruning_violation_proof_of_work_and_difficulty.go):

- Validates that blocks which cause finality violations (reorgs past the finality point) require extra proof-of-work
- This protects against deep reorganization attacks

---

## Block Insertion Flow

The complete block insertion flow is orchestrated by `blockProcessor.ValidateAndInsertBlock` at [validate_and_insert_block.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/processes/blockprocessor/validate_and_insert_block.go):

```
1. Create staging area
2. Validate block header in isolation
3. Resolve block parents via BlockParentBuilder
4. Set DAG topology relations
5. Run GhostDAG to compute blue score, blue work, merge sets
6. Validate block header in context
7. If header-only: store header and return StatusHeaderOnly
8. Validate block body in isolation
9. Stage block acceptance data
10. Add block to reachability tree
11. Calculate and stage DAA score and difficulty
12. Update UTXO diff
13. Validate block body in context (UTXO operations)
14. Update virtual state
15. Compute finality point
16. Update header selected tip
17. Commit all staged changes to database
```

### Virtual Resolution

After block insertion, if `updateVirtual` is `true`, the virtual block is resolved. This happens in chunks of 100 blocks to prevent holding the consensus lock for too long:

```go
const virtualResolveChunk = 100
```

The `ResolveVirtual` method iteratively resolves blocks until the virtual is fully synced with all known blocks. It reports progress via a callback.

### Trusted Data Variant

`ValidateAndInsertBlockWithTrustedData` accepts blocks that include **trusted GhostDAG data** and **DAA window data** from peers. This is used during IBD (Initial Block Download) to skip re-computing GhostDAG data, dramatically accelerating sync.

---

## UTXO Model

Eiyaro uses a **UTXO-based transaction model** similar to Bitcoin, but with DAG-specific enhancements.

### UTXOEntry

```go
type UTXOEntry interface {
    Amount() uint64
    ScriptPublicKey() *ScriptPublicKey
    BlockDAAScore() uint64
    IsCoinbase() bool
    Equal(other UTXOEntry) bool
}
```

### UTXO Diff Algebra

Instead of storing the full UTXO set for every block, Eiyaro stores **UTXO diffs** — the additions and removals caused by each block. The virtual UTXO set is reconstructed by applying diffs along the selected parent chain.

This is managed by `ConsensusStateManager` which:
- Maintains UTXO diff child chains
- Restores UTXO sets from acceptance data
- Handles UTXO diff re-orgs efficiently

### Multiset Commitment

Each block header contains a `UTXOCommitment` — a multiset hash of the UTXO set at that block. This allows light clients to verify the UTXO set without downloading it. The `MultisetStore` tracks these hashes.

---

## Coinbase and Block Rewards

### Block Subsidy

The `CoinbaseManager` at [coinbasemanager.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/processes/coinbasemanager/coinbasemanager.go) computes the block subsidy:

1. **Genesis**: Special genesis reward (`SubsidyGenesisReward`)
2. **Pre-Deflationary Phase**: Constant base subsidy (8 EY = `8 * SeyPerEY`) until DAA score reaches `defaultDeflationaryPhaseDaaScore` (360)
3. **Deflationary Phase**: Subsidy decays according to a pre-computed table, reducing by approximately 10% annually (using a curve factor of 1)

The subsidy is combined with transaction fees to form each block's total reward.

### Coinbase Transaction

The coinbase transaction is created with:
- One output per blue block in the merge set (paying the miner of each blue block their reward + fees)
- A red reward output (remaining fees from red blocks going to the merging miner)
- For block version >= 2: a development fee output (1% dev fee)
- A payload containing blue score, script public key, extra data, and subsidy

---

## Difficulty Adjustment (DAA)

The `DifficultyManager` at [difficultymanager.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/processes/difficultymanager/difficultymanager.go) implements the difficulty adjustment algorithm.

### Window-Based Adjustment

The DAA uses a window of the last `difficultyAdjustmentWindowSize` blocks (default: 2641) to compute the required difficulty:

```
newTarget = averageWindowTarget * (windowMaxTimestamp - windowMinTimestamp) / (targetTimePerBlock * windowSize)
```

### DAA Score

The **DAA Score** is computed as:
```
daaScore = selectedParentDAAScore + len(DAAAddedBlocks)
```

Where `DAAAddedBlocks` are those blocks in the merge set that also appear in the DAA window. This ensures that only blocks within the difficulty window contribute to the DAA score.

### Parameters

- **Target Time Per Block**: 20 seconds (Eiyaro)
- **Difficulty Adjustment Window**: 2641 blocks
- **PowMax**: 2^255 - 1
- **DisableDifficultyAdjustment**: Can be set to `true` for test networks (uses genesis bits)

---

## Finality

The `FinalityManager` at [finality_manager.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/processes/finalitymanager/finality_manager.go) determines block finality in the GhostDAG protocol.

### Finality Point

The **finality point** of a block is the newest block in its selected parent chain whose blue score difference from the block is at least `finalityDepth`:

```
requiredBlueScore = block.BlueScore() - finalityDepth
finalityPoint = block in selected parent chain with highest blue score < requiredBlueScore
```

### Finality Depth

Default: `24 hours / 20 seconds = 4320` blue score units. This means a transaction is considered final when it is buried under approximately 24 hours worth of blue blocks.

### Finality Violations

A finality violation occurs when a block causes a reorganization beyond the finality point. Such blocks are rejected unless they provide additional proof-of-work (checked by `PruningViolationProofOfWorkAndDifficulty`).

### Key Methods

- `VirtualFinalityPoint(stagingArea)`: Returns the finality point from the virtual block's perspective
- `FinalityPoint(stagingArea, blockHash, isBlockWithTrustedData)`: Returns the finality point for a specific block
- `calculateAndStageFinalityPoint`: Computes and caches the finality point

---

## Pruning

The `PruningManager` at [pruningmanager.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/processes/pruningmanager/pruningmanager.go) manages the pruning of old block data.

### Pruning Point

The **pruning point** is the oldest block for which the full UTXO set is maintained. Blocks older than the pruning point are pruned to **header-only** status (body deleted, header retained).

### Pruning Point Advancement

The pruning point advances when:
1. A new candidate is found at sufficient depth (`pruningDepth` blue score units behind virtual)
2. The candidate's finality score exceeds the current pruning point's finality score
3. On archival nodes, block bodies are never deleted

### Pruning Depth

```go
pruningDepth = 2*finalityDepth + 4*MergeSetSizeLimit*K + 2*K + 2
```

Default: approximately 2 days worth of blue blocks.

### Data Deletion

When pruning:
1. Blocks in the pruning point's past and anticone are identified
2. Blocks in the pruning point and its anticone are kept (needed for future block validation)
3. All other blocks below the pruning point are marked `StatusHeaderOnly`
4. On non-archival nodes, block bodies, acceptance data, multisets, UTXO diffs, and DAA data are deleted

### Data Retention and Pruning Interval

- `DataRetentionDuration`: Minimum age of data before pruning deletion (e.g., 24h for full nodes)
- `PruningInterval`: Minimum time between pruning deletions
- `DeletionDepth`: Number of pruning point indices to retain before deletion (extra safety margin)

### Pruning UTXO Set

Each pruning point maintains its own UTXO set, stored in `PruningStore`. This allows syncing peers to bootstrap from any pruning point.

---

## Pruning Proofs

The `PruningProofManager` at [pruningproofmanager.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/processes/pruningproofmanager/pruningproofmanager.go) generates and validates **pruning proofs** for header-first IBD.

### Purpose

Pruning proofs allow a syncing node to verify that a claimed pruning point is valid without downloading all intermediate blocks. The proof contains block headers at multiple levels that connect the pruning point's parents to the selected tip.

### Structure

```go
type PruningPointProof struct {
    Headers [][]BlockHeader  // One slice per block level
}
```

### Building a Proof

1. Start from the current pruning point
2. For each block level, find the block at depth `2*pruningProofM` from the selected tip
3. Collect all headers in the selected tip's chain that are ancestors of the root block
4. Cross-link levels by ensuring the block at depth `pruningProofM` at level N+1 connects to level N

### Validation

The validator:
1. Reconstructs a temporary DAG from the proof headers
2. Runs GhostDAG on each level independently
3. Verifies that the selected tip at each level is a parent of the pruning point
4. Checks that the proof's blue work is competitive with the current DAG state

### Application

`ApplyPruningPointProof` stores all proof headers, computes GhostDAG data, sets finality points, and initializes consensus state — enabling header-first sync.

---

## Reachability

The `ReachabilityManager` at [reachabilitymanager.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/processes/reachabilitymanager/reachabilitymanager.go) provides sub-linear time queries for block ancestry.

### Reachability Tree

Each block is assigned an **interval** in a reachability tree, keyed by its selected parent chain position. A block A is an ancestor of block B if A's interval contains B's interval:

- `IsDAGAncestorOf(A, B)`: True if A is in B's DAG past
- `IsReachabilityTreeAncestorOf(A, B)`: True if A is in B's selected parent chain

### Future Covering Set

Each block maintains a **future covering set** — the set of merge set red/blue blocks that are in its future but not in its selected parent chain. This enables efficient anticone queries.

### Reindexing

When the selected tip changes, the reachability tree is reindexed to maintain efficient queries. The `reindexSlack` and `reindexWindow` parameters control this process.

---

## Sync Management

The `SyncManager` at [syncmanager.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/processes/syncmanager/syncmanager.go) handles peer synchronization.

### Key Operations

- **GetHashesBetween(lowHash, highHash, maxBlocks)**: Returns block hashes in the selected parent chain between two hashes. Uses antiPast traversal to efficiently skip irrelevant blocks.

- **GetAnticone(blockHash, contextHash, maxBlocks)**: Returns blocks in the anticone of `blockHash` from the perspective of `contextHash`. Used during IBD to request missing blocks.

- **GetMissingBlockBodyHashes(highHash)**: Returns hashes of blocks whose bodies are missing (header-only) in the selected parent chain.

- **CreateBlockLocator(lowHash, highHash, limit)**: Creates a block locator (sparse list of hashes) for chain comparison.

- **CreateHeadersSelectedChainBlockLocator**: Creates a locator for the headers-selected chain (used by header-first nodes).

- **GetSyncInfo()**: Returns current sync state including whether the node is synced.

---

## Consensus Interface Methods

The full `Consensus` interface at [consensus.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/model/externalapi/consensus.go#L4-L63):

### Initialization
```go
Init(skipAddingGenesis bool) error
```
Initializes the consensus database. If `skipAddingGenesis` is false and the database is empty, the genesis block is automatically inserted.

### Block Building
```go
BuildBlock(coinbaseData *DomainCoinbaseData, transactions []*DomainTransaction) (*DomainBlock, error)
```
Builds a new block template from the current virtual state. Returns the block and whether it includes a red reward.

```go
BuildBlockTemplate(coinbaseData *DomainCoinbaseData, transactions []*DomainTransaction) (*DomainBlockTemplate, error)
```
Like `BuildBlock` but returns additional metadata: `CoinbaseHasRedReward` and `IsNearlySynced`.

### Block Validation and Insertion
```go
ValidateAndInsertBlock(block *DomainBlock, updateVirtual bool, powSkip bool) error
```
Validates and inserts a block into the DAG. If `updateVirtual` is true, also resolves the virtual block. If `powSkip` is true, skips proof-of-work validation.

```go
ValidateAndInsertBlockWithTrustedData(block *BlockWithTrustedData, validateUTXO bool) error
```
Inserts a block with pre-computed GhostDAG and DAA window data from a trusted peer (IBD path).

```go
ValidateTransactionAndPopulateWithConsensusData(transaction *DomainTransaction) error
```
Validates a transaction against the virtual state and populates its UTXO entries and fee.

### Block Queries
```go
GetBlock(blockHash *DomainHash) (*DomainBlock, bool, error)
```
Returns the full block (header + transactions). Second return value indicates existence.

```go
GetBlockEvenIfHeaderOnly(blockHash *DomainHash) (*DomainBlock, error)
```
Returns the block even if only the header is available (body may be pruned). Falls back to `BlockHeaderStore` if not found in `BlockStore`.

```go
GetBlockHeader(blockHash *DomainHash) (BlockHeader, error)
```
Returns just the block header.

```go
GetBlockHeaders(blockHashes []*DomainHash) ([]BlockHeader, error)
```
Batch header retrieval — significantly more efficient than repeated `GetBlockHeader` calls.

```go
HasBlock(blockHash *DomainHash) (bool, error)
```
Checks if a block exists in the store.

```go
GetBlockInfo(blockHash *DomainHash) (*BlockInfo, error)
```
Returns block metadata: existence, status, blue score, blue work, selected parent, merge set blues and reds.

```go
GetBlockRelations(blockHash *DomainHash) (parents, children []*DomainHash, err error)
```
Returns the DAG parents and children of a block.

```go
GetBlockAcceptanceData(blockHash *DomainHash) (AcceptanceData, error)
```
Returns the acceptance data for a block (which transactions were accepted).

```go
GetBlocksAcceptanceData(blockHashes []*DomainHash) ([]AcceptanceData, error)
```
Batch acceptance data retrieval.

```go
GetBlockByTransactionID(transactionID *DomainTransactionID) (*DomainBlock, error)
```
Scans all stored blocks to find the block containing a given transaction.

### Chain Queries
```go
GetHashesBetween(lowHash, highHash *DomainHash, maxBlocks uint64) ([]*DomainHash, *DomainHash, error)
```
Returns block hashes in the selected parent chain between two hashes, up to `maxBlocks`.

```go
GetAnticone(blockHash, contextHash *DomainHash, maxBlocks uint64) ([]*DomainHash, error)
```
Returns blocks in the anticone of `blockHash` from `contextHash`'s perspective.

```go
GetMissingBlockBodyHashes(highHash *DomainHash) ([]*DomainHash, error)
```
Returns hashes of blocks whose bodies are missing in the selected parent chain.

```go
IsInSelectedParentChainOf(blockHashA, blockHashB *DomainHash) (bool, error)
```
Returns true if `blockHashA` is in the selected parent chain of `blockHashB`.

```go
IsChainBlock(blockHash *DomainHash) (bool, error)
```
Returns true if the block is in the virtual's selected parent chain.

```go
GetVirtualSelectedParentChainFromBlock(blockHash *DomainHash) (*SelectedChainPath, error)
```
Returns the selected chain path from a block to the virtual.

### Virtual State
```go
GetVirtualInfo() (*VirtualInfo, error)
```
Returns virtual block info: parent hashes, required difficulty, past median time, blue score, DAA score.

```go
GetVirtualSelectedParent() (*DomainHash, error)
```
Returns the virtual block's selected parent hash.

```go
GetVirtualDAAScore() (uint64, error)
```
Returns the virtual block's DAA score.

```go
GetVirtualUTXOs(expectedVirtualParents []*DomainHash, fromOutpoint *DomainOutpoint, limit int) ([]*OutpointAndUTXOEntryPair, error)
```
Returns virtual UTXOs, paginated. Validates that the caller's view of virtual parents matches.

```go
ResolveVirtual(progressReportCallback func(uint64, uint64)) error
```
Fully resolves the virtual block. The callback is invoked periodically with (startDAAScore, currentDAAScore) for progress reporting.

```go
Tips() ([]*DomainHash, error)
```
Returns the current DAG tips.

```go
Anticone(blockHash *DomainHash) ([]*DomainHash, error)
```
Returns the anticone of a block from the tips' perspective.

### Pruning
```go
PruningPoint() (*DomainHash, error)
```
Returns the current pruning point hash.

```go
PruningPointHeaders() ([]BlockHeader, error)
```
Returns headers of all historical pruning points.

```go
PruningPointAndItsAnticone() ([]*DomainHash, error)
```
Returns the pruning point and all blocks in its anticone (from the virtual block's perspective). Results are cached.

```go
GetPruningPointUTXOs(expectedPruningPointHash *DomainHash, fromOutpoint *DomainOutpoint, limit int) ([]*OutpointAndUTXOEntryPair, error)
```
Returns UTXOs at the pruning point, paginated. Validates the expected pruning point hash.

```go
IsValidPruningPoint(blockHash *DomainHash) (bool, error)
```
Checks if a block could serve as a valid pruning point.

```go
ArePruningPointsViolatingFinality(pruningPoints []BlockHeader) (bool, error)
```
Checks if a set of pruning points from a peer would cause a finality violation.

```go
ClearImportedPruningPointData() error
```
Clears temporary data from an aborted pruning point import.

```go
AppendImportedPruningPointUTXOs(outpointAndUTXOEntryPairs []*OutpointAndUTXOEntryPair) error
```
Appends UTXOs during a pruning point IBD import.

```go
ValidateAndInsertImportedPruningPoint(newPruningPoint *DomainHash) error
```
Validates and activates an imported pruning point.

```go
ImportPruningPoints(pruningPoints []BlockHeader) error
```
Imports a set of historical pruning points.

### Pruning Proofs
```go
BuildPruningPointProof() (*PruningPointProof, error)
```
Builds a pruning proof for the current pruning point.

```go
ValidatePruningPointProof(pruningPointProof *PruningPointProof) error
```
Validates a pruning proof from a peer (used during IBD).

```go
ApplyPruningPointProof(pruningPointProof *PruningPointProof) error
```
Applies a validated pruning proof to the consensus database. **Not atomic** — if interrupted, the database must be discarded.

### Block Locators
```go
CreateBlockLocatorFromPruningPoint(highHash *DomainHash, limit uint32) (BlockLocator, error)
```
Creates a block locator from the pruning point to `highHash`.

```go
CreateHeadersSelectedChainBlockLocator(lowHash, highHash *DomainHash) (BlockLocator, error)
```
Creates a locator for the headers-selected chain.

```go
CreateFullHeadersSelectedChainBlockLocator() (BlockLocator, error)
```
Creates a locator from the pruning point to the headers-selected tip.

### Sync and Network
```go
GetSyncInfo() (*SyncInfo, error)
```
Returns current sync status information.

```go
IsNearlySynced() (bool, error)
```
Returns true if the virtual selected parent's timestamp is within the expected DAA window duration, indicating the node is close to being synced. Eiyaro extends this to always return true when only the genesis block exists (for isolated test nodes).

```go
GetHeadersSelectedTip() (*DomainHash, error)
```
Returns the current header-only selected tip.

### Difficulty and Mining
```go
EstimateNetworkHashesPerSecond(startHash *DomainHash, windowSize int) (uint64, error)
```
Estimates the network hashrate over a window starting from `startHash`.

```go
PopulateMass(transaction *DomainTransaction)
```
Calculates and sets the mass of a transaction.

### DAA and Trusted Data
```go
BlockDAAWindowHashes(blockHash *DomainHash) ([]*DomainHash, error)
```
Returns the DAA window block hashes for a given block.

```go
TrustedDataDataDAAHeader(trustedBlockHash, daaBlockHash *DomainHash, daaBlockWindowIndex uint64) (*TrustedDataDataDAAHeader, error)
```
Returns the GhostDAG data and header for a DAA window block used in trusted data.

```go
TrustedBlockAssociatedGHOSTDAGDataBlockHashes(blockHash *DomainHash) ([]*DomainHash, error)
```
Returns the K+1 selected parent chain hashes associated with a trusted block's GhostDAG data.

```go
TrustedGHOSTDAGData(blockHash *DomainHash) (*BlockGHOSTDAGData, error)
```
Returns GhostDAG data for a block, falling back to trusted data if the regular data is not available.

### Merge Depth
```go
VirtualMergeDepthRoot() (*DomainHash, error)
```
Returns the current virtual merge depth root.

---

## Network Selection

Eiyaro supports multiple network types, each with different parameters. The defaults are defined in [consensus_defaults.go](file:///d:/Eiyaro/Eiyaro/domain/dagconfig/consensus_defaults.go):

| Parameter | Value |
|-----------|-------|
| GhostDAG K | 18 |
| Target block time | 20 seconds |
| Difficulty window | 2641 blocks |
| Max block mass | 500,000 grams |
| Max block parents | 10 |
| Merge set size limit | 180 (K * 10) |
| Finality duration | 24 hours (~4320 blue score) |
| Timestamp deviation tolerance | 132 |
| Genesis subsidy | 8 EY (`8 * SeyPerEY`) |
| Pre-deflationary base subsidy | 8 EY |
| Deflationary base subsidy | 8 EY |
| Deflationary curve factor | 1.0 |
| Pruning proof M | 1000 |
| Max block level | Configurable per network |
| Merge depth | 360 |

The `Factory.Config` struct at [factory.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/factory.go#L72-L94) allows overriding any parameter:

```go
type Config struct {
    dagconfig.Params
    IsArchival                      bool
    EnableSanityCheckPruningUTXOSet bool
    SkipAddingGenesis               bool
    DeletionDepth                   uint64
    DataRetentionDuration           time.Duration
    PruningInterval                 time.Duration
    UseHoohashCLibrary              bool
}
```

---

## Usage Examples

### Creating a Consensus Instance

```go
import (
    "github.com/Eiyaro/Eiyaro/domain/consensus"
    "github.com/Eiyaro/Eiyaro/domain/dagconfig"
    "github.com/Eiyaro/Eiyaro/domain/prefixmanager/prefix"
    "github.com/Eiyaro/Eiyaro/infrastructure/db/database/ldb"
)

// Open database
db, err := ldb.NewLevelDB("/data/eiyaro", 4096)

// Create factory
factory := consensus.NewFactory()

// Configure for mainnet
config := &consensus.Config{
    Params:                          dagconfig.MainnetParams.Params,
    IsArchival:                      false,
    EnableSanityCheckPruningUTXOSet: true,
}

// Create consensus instance
consensusInstance, shouldMigrate, err := factory.NewConsensus(
    config, db, &prefix.Prefix{}, nil,
)
```

### Validating and Inserting a Block

```go
block := &externalapi.DomainBlock{
    Header: header,
    Transactions: txs,
}

// Insert block and update virtual
err = consensusInstance.ValidateAndInsertBlock(block, true, false)
if err != nil {
    // Block was rejected — check ruleerrors type
    log.Errorf("block rejected: %v", err)
}
```

### Querying Block Information

```go
// Get block info
info, err := consensusInstance.GetBlockInfo(blockHash)
fmt.Printf("Blue score: %d\n", info.BlueScore)
fmt.Printf("Status: %s\n", info.BlockStatus)

// Get full block
block, exists, err := consensusInstance.GetBlock(blockHash)

// Check chain membership
isChainBlock, err := consensusInstance.IsChainBlock(blockHash)
```

### Syncing the Virtual

```go
err = consensusInstance.ResolveVirtual(func(startDAAScore, currentDAAScore uint64) {
    log.Infof("Resolving virtual: %d / %d", currentDAAScore, startDAAScore)
})
```

---

## Concurrency Model

The consensus engine uses a **single global mutex** (`sync.Mutex`) for all state-modifying operations. This ensures that:

1. Block insertion is **single-threaded** — only one block can be inserted at a time
2. Virtual resolution chunks temporarily release the lock to prevent starvation
3. Read operations (like `GetBlock`, `GetBlockInfo`) also hold the lock briefly

The lock is managed transparently — every exported method acquires and releases it automatically.

### Virtual Resolution and Lock Release

During `ValidateAndInsertBlock` with `updateVirtual=true`, the virtual resolution runs in chunks of 100 blocks. Between chunks, the lock is released to allow other operations (like new block arrivals) to interleave:

```go
const virtualResolveChunk = 100  // Must be smaller than FinalityDuration

for {
    s.lock.Unlock()
    // ... other operations can run here ...
    s.lock.Lock()
    _, isCompletelyResolved, err := s.resolveVirtualChunkNoLock(virtualResolveChunk)
    if isCompletelyResolved {
        break
    }
}
```

---

## Events and Notifications

Consensus emits events through the `consensusEventsChan` (a channel of `ConsensusEvent`):

- **BlockAdded**: Emitted when a non-header-only block is successfully inserted. Contains the full block.

- **VirtualChangeSet**: Emitted when the virtual state changes after resolution. Contains the new virtual parents, the virtual selected parent's blue score, and the new virtual DAA score.

The channel is bounded; if full, events are dropped with an error. This prevents consensus from blocking on slow event consumers.

---

## Database and Staging

### Staging Area Pattern

All state mutations use a **staging area** pattern to ensure atomicity:

```go
stagingArea := model.NewStagingArea()

// Stage changes to stores
store.Stage(stagingArea, key, value)

// Commit all changes atomically
err := staging.CommitAllChanges(databaseContext, stagingArea)
```

If any step fails, the staging area is discarded and no changes are persisted. This is critical because block validation is multi-step — partial database updates would corrupt the consensus state.

### Storage Architecture

Data is stored in LevelDB with prefix-based key namespaces. Each store type has its own prefix, allowing multiple consensus instances to share a single LevelDB database by using different top-level prefixes.

### Cache Sizing

Cache sizes are configured based on pruning depth and finality depth, with a `largeCacheDivisor` (environment variable `HTND_LARGE_CACHE_DIVISOR`) that divides the cache sizes for memory-constrained environments.

---

## Block Status Lifecycle

A block's status transitions through several states ([blockstatus.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/model/externalapi/blockstatus.go)):

```
                    ┌──────────────────┐
                    │   StatusInvalid   │  ← Validation failed
                    └──────────────────┘
                    
New Block ──► Validate Header ──► StatusHeaderOnly  ← Header-only (body unknown)
                    │
                    ▼ Validate Body
           StatusUTXOPendingVerification  ← Valid but not yet in selected parent chain
                    │
                    ▼
              StatusUTXOValid  ← Fully validated, UTXO operations verified
                    
              StatusDisqualifiedFromChain  ← Not eligible as selected parent
```

- `StatusInvalid (0)`: Block failed validation — permanently rejected
- `StatusUTXOValid (1)`: Block is fully valid with UTXO operations verified
- `StatusUTXOPendingVerification (2)`: Block valid but pending UTXO verification (not yet in selected parent chain, or violates finality)
- `StatusDisqualifiedFromChain (3)`: Block not eligible to be a selected parent
- `StatusHeaderOnly (4)`: Only the header exists — body is pruned or not yet received

After pruning, a block's status is set back to `StatusHeaderOnly` and its body data is deleted.

---

## Notes & Caveats

1. **Single-threaded insertion**: The global mutex means only one block insertion can proceed at a time. This is by design to ensure deterministic ordering.

2. **Database initialization**: The `Init` method must be called before using consensus. The `Factory` handles this automatically.

3. **Memory requirements**: Large DAGs require significant memory for caches. The `HTND_LARGE_CACHE_DIVISOR` environment variable can reduce cache sizes.

4. **Archival vs. pruning nodes**: Archival nodes (`IsArchival=true`) retain all block data forever. Non-archival nodes prune old blocks to header-only.

5. **Pruning proof non-atomicity**: `ApplyPruningPointProof` is explicitly non-atomic. If interrupted, the database for that consensus instance must be recreated.

6. **Virtual resolution batching**: Virtual resolution processes at most `virtualResolveChunk` (100) blocks per lock acquisition to prevent lock starvation.

7. **Lock contention**: Long-running operations like `GetBlockByTransactionID` release and re-acquire the lock periodically to avoid blocking other operations.

8. **GhostDAG determinism**: The algorithm is fully deterministic given the same DAG state. Tie-breaking uses lexicographic hash comparison.

9. **UTXO diff chains**: The UTXO diff child chain mechanism requires periodic validation. `ValidateUTXODiffChildChains` can repair broken chains.

10. **Finality and pruning synchronization**: The pruning point cannot advance faster than one finality interval at a time (for block versions < 5), preventing rapid pruning point attacks.

---

## FAQ

### How long does it take for a transaction to be considered final?

Finality is determined by blue score depth, not clock time. With `FinalityDuration = 24h` and `TargetTimePerBlock = 20s`, the computed `FinalityDepth` is 4,320 blue score units. In practice, a transaction is final when it is buried under approximately 24 hours worth of blue blocks. Blocks that attempt to reorganize past the finality point are rejected unless they provide additional proof-of-work.

### What happens during a chain reorganization?

When a new block's selected parent chain differs from the current virtual's selected parent chain, a reorganization occurs. The consensus engine emits a `VirtualChangeSet` event containing the new virtual parents and DAA score. The `VirtualSelectedParentChainChanged` notification provides the removed and added chain block hashes. UTXO diffs are reapplied along the new selected parent chain, ensuring the virtual UTXO set reflects the heaviest chain.

### How does Eiyaro prevent double-spending?

Each transaction input references a previous UTXO via its outpoint. During block body validation, the consensus engine checks that each UTXO being spent exists and has not already been consumed by another transaction in the same block or in a previously accepted block. The GhostDAG blue/red classification ensures that conflicting transactions in different blocks are resolved deterministically — only transactions in blue blocks are considered confirmed, while those in red blocks are orphaned.

### What is a pruning point and why does it matter?

The pruning point is the oldest block for which full UTXO set data is retained. Blocks older than the pruning point are pruned to header-only status, deleting their transaction bodies, acceptance data, and UTXO diffs. This keeps storage requirements bounded while retaining enough history for block validation. The pruning depth is approximately 2 days of blue blocks, computed as `2×finalityDepth + 4×MergeSetSizeLimit×K + 2×K + 2`.

### Why is there a global mutex on block insertion?

Block insertion is intentionally single-threaded via a global `sync.Mutex` to guarantee deterministic ordering of state transitions. Only one block can be inserted at a time, and virtual resolution processes blocks in chunks of 100, releasing the lock between chunks to prevent starvation. This design ensures that all nodes converge on the same consensus state regardless of the order in which blocks arrive.

### What does a block's Blue Score represent?

Blue Score is the primary chain-weight metric in GhostDAG, analogous to Bitcoin's chain height. It is computed as the selected parent's blue score plus the number of merge set blue blocks (including the selected parent itself). The Blue Work is a related metric that accumulates actual proof-of-work: selected parent's blue work plus the work of each blue block in the merge set. The selected parent chain follows the path of highest blue work, with lexicographic hash comparison for tie-breaking.