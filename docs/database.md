# Eiyaro Database Layer

## Overview

Eiyaro uses an embedded key-value store for persistent storage of all blockchain data. The database layer provides a transaction-based interface with cursor-based iteration, supporting two interchangeable backends: **LevelDB** (the default) and **PebbleDB** (CockroachDB's next-generation storage engine).

The architecture follows a clean layered design:

```
Layer 4: Consensus Stores (block store, UTXO store, header store, etc.)
Layer 3: DBManager        (consensus-domain wrapper: model.DBManager)
Layer 2: database         (generic KV interface: Database, Transaction, Cursor)
Layer 1: Backend          (ldb.LevelDB / pebble.DB)
```

Each layer is separated by interfaces, enabling backend portability and testability. The raw backend is never accessed directly by consensus code — all access goes through `model.DBManager`, which delegates to the generic `database.Database` interface.

---

## Backend Options

### LevelDB

**Package:** `github.com/Eiyaro/Eiyaro/infrastructure/db/database/ldb`

LevelDB is the default backend, built on Google's [syndtr/goleveldb](https://github.com/syndtr/goleveldb). It is a battle-tested LSM-tree storage engine with the following characteristics:

- **Compression:** Snappy (configured in `ldb/options.go`)
- **Bloom filter:** 16 bits/key (configurable via `HTND_BLOOM_FILTER_LEVEL` or `BLOOM_FILTER_LEVEL` environment variable)
- **Write buffer:** 64 MiB by default
- **Block cache:** 1024 MiB by default
- **Open files cache:** 1024 file descriptors
- **Compaction table size:** 32 MiB
- **Compaction total size:** 1024 MiB
- **L0 compaction trigger:** 8 (configurable via `KSDB_COMPACTION_L0_TRIGGER`)
- **Write L0 slowdown trigger:** 24 (configurable via `KSDB_WRITE_L0_SLOWDOWN`)
- **Write L0 pause trigger:** 48 (configurable via `KSDB_WRITE_L0_PAUSE`)

**Corruption recovery:** On open, if LevelDB detects corruption (`ErrCorrupted`), it automatically attempts recovery via `leveldb.RecoverFile()`.

**Constructor:**

```go
func NewLevelDB(path string, cacheSizeMiB int) (*LevelDB, error)
```

The `cacheSizeMiB` parameter, when > 0, overrides both `BlockCacheCapacity` (set to `cacheSizeMiB * opt.MiB`) and `WriteBuffer` (set to half of that). If `cacheSizeMiB` is 0, the defaults from `Options()` are used.

### PebbleDB

**Package:** `github.com/Eiyaro/Eiyaro/infrastructure/db/database/pebble`

PebbleDB is an alternative backend from [CockroachDB/pebble](https://github.com/cockroachdb/pebble), inspired by RocksDB. It offers **better write performance** and is enabled via environment variable configuration.

Key features over LevelDB:

- **Newest format version** (`pebble.FormatNewest`) — enables columnar blocks and value separation
- **Value separation** — blobs larger than 1 KiB are stored in separate files, reducing write amplification
- **Per-level compression control** — L0-L3 use no compression, L4-L6 use Snappy
- **Read-triggered compaction** — hot data gets compacted more aggressively (256 MiB read compaction rate)
- **Log-based event listener** — optional detailed logging of compactions, flushes, write stalls, and disk slowdowns
- **Cursor tracking** — the DB tracks all open cursors and closes them automatically on DB close

**Corruption recovery:** On open, if Pebble detects corruption (`pebble.ErrCorruption`), the entire database directory is removed (`os.RemoveAll`) and a fresh DB is created. This is a nuclear recovery strategy — all data is lost and the node will need to resync from the network.

**Constructor:**

```go
func NewPebbleDB(path string, cacheSizeMiB int) (*DB, error)
```

The `cacheSizeMiB` parameter, when > 0, sets the block cache size. If 0, defaults to 4096 MiB.

---

## Database Interface

**Package:** `github.com/Eiyaro/Eiyaro/infrastructure/db/database`

The core `Database` interface is defined in [database.go](file:///d:/Eiyaro/Eiyaro/infrastructure/db/database/database.go). It embeds `DataAccessor` and adds lifecycle methods:

```go
type Database interface {
    DataAccessor

    // Begin begins a new database transaction.
    Begin() (Transaction, error)

    // Compact compacts the database instance.
    Compact() error

    // Close closes the database.
    Close() error
}
```

The separation of `Database` from `DataAccessor` is intentional: since `Transaction` also embeds `DataAccessor`, merging them would force transaction implementors to implement `Begin()` and `Close()`, which is undesirable.

---

## DataAccessor Interface

Defined in [dataaccessor.go](file:///d:/Eiyaro/Eiyaro/infrastructure/db/database/dataaccessor.go), this is the core CRUD interface shared by both `Database` and `Transaction`:

```go
type DataAccessor interface {
    Put(key *Key, value []byte) error
    BatchPut(pairs map[*Key][]byte) error
    Get(key *Key) ([]byte, error)
    Has(key *Key) (bool, error)
    Delete(key *Key) error
    Cursor(bucket *Bucket) (Cursor, error)
}
```

### Method Semantics

| Method | Description |
|--------|-------------|
| `Put(key, value)` | Sets the value for the given key. Overwrites any previous value. |
| `BatchPut(pairs)` | Atomically sets multiple key-value pairs. Implemented via LevelDB `Batch` or Pebble `Batch`. |
| `Get(key)` | Retrieves the value for the given key. Returns `ErrNotFound` if the key does not exist. |
| `Has(key)` | Returns `true` if the key exists in the database. |
| `Delete(key)` | Removes the key. Does not return an error if the key does not exist. |
| `Cursor(bucket)` | Opens a new cursor positioned over the given bucket prefix. |

**Important:** All methods operate on the key space prefixed by the bucket path contained in the `*Key` argument.

---

## Transaction Interface

Defined in [transaction.go](file:///d:/Eiyaro/Eiyaro/infrastructure/db/database/transaction.go):

```go
type Transaction interface {
    DataAccessor

    // Rollback rolls back whatever changes were made to the
    // database within this transaction.
    Rollback() error

    // Commit commits whatever changes were made to the database
    // within this transaction.
    Commit() error

    // RollbackUnlessClosed rolls back changes that were made to
    // the database within the transaction, unless the transaction
    // had already been closed using either Rollback or Commit.
    RollbackUnlessClosed() error
}
```

### Transaction Semantics

> **⚠️ Important:** Transactions provide data consistency over the state of the database **as it was when the transaction started**. There is **NO guarantee** that if one puts data into the transaction then it will be available to get within the same transaction.

### Implementation Differences

#### LevelDB Transaction

- Builds a `leveldb.Batch` in memory
- `Put` and `Delete` are buffered in the batch; they only take effect on `Commit()`
- `Get` and `Has` read directly from the underlying database (**not** from the batch)
- `Cursor` delegates to the database's cursor
- `Rollback()` resets the batch; `Commit()` writes the batch via `db.ldb.Write()`

#### Pebble Transaction

- Uses a `pebble.NewIndexedBatch()` — an **indexed batch** that supports reads of uncommitted modifications
- Tracks key modifications in a `keyModifications` map for `Has()` support within the transaction
- `Get()` checks the indexed batch first, then falls back to the database
- `Has()` checks `keyModifications` first, then falls back to the database
- Tracks all opened cursors and closes them on `Commit()` / `Rollback()`

### Usage Pattern

```go
tx, err := db.Begin()
if err != nil {
    return err
}
defer tx.RollbackUnlessClosed()

tx.Put(bucket.Key(suffix), value)
// ... more operations ...

return tx.Commit()
```

---

## Cursor Interface

Defined in [cursor.go](file:///d:/Eiyaro/Eiyaro/infrastructure/db/database/cursor.go):

```go
type Cursor interface {
    Next() bool
    First() bool
    Seek(key *Key) error
    Key() (*Key, error)
    Value() ([]byte, error)
    Close() error
}
```

### Method Semantics

| Method | Description |
|--------|-------------|
| `Next()` | Moves to the next key/value pair. Returns `false` if exhausted. Panics if closed. |
| `First()` | Moves to the first key/value pair in the bucket. Returns `false` if bucket is empty. Panics if closed. |
| `Seek(key)` | Seeks to the first key >= the given key. Returns `ErrNotFound` if no such key exists. |
| `Key()` | Returns the current key (trimmed of the bucket prefix). Returns `ErrNotFound` if exhausted. Caller must not modify the returned data. |
| `Value()` | Returns the current value. Returns `ErrNotFound` if exhausted. Caller must not modify the returned data. |
| `Close()` | Closes the cursor and releases resources. Must be called or resources leak. |

### Key Trimming

The cursor is opened on a bucket prefix (e.g., `blocks/`). The `Key()` method automatically strips the bucket prefix, returning only the suffix portion. This means callers only work with the sub-key within the bucket namespace.

### Pebble Cursor: First-Page-On-Next Optimization

The Pebble cursor implements a specific optimization: if `Next()` is called before `First()` or `Seek()`, it automatically calls `First()` internally. This addresses the Pebble iterator behavior where `Next()` on a freshly created (unpositioned) iterator behaves unexpectedly.

### Cursor Lifecycle

- LevelDB cursors: Tracked by `isClosed` flag; panic on use-after-close
- Pebble cursors: Registered with the parent `DB` via `registerCursor`/`deregisterCursor`; automatically closed when the DB is closed

### Usage Pattern

```go
cursor, err := db.Cursor(bucket)
if err != nil {
    return err
}
defer cursor.Close()

for ok := cursor.First(); ok; ok = cursor.Next() {
    key, _ := cursor.Key()
    value, _ := cursor.Value()
    // process key/value
}
```

---

## Key Layout

**Package:** `github.com/Eiyaro/Eiyaro/infrastructure/db/database` in [keys.go](file:///d:/Eiyaro/Eiyaro/infrastructure/db/database/keys.go)

### Key Structure

A `Key` is composed of a bucket path (prefix) concatenated with a suffix:

```
|---- bucket path ----|---- suffix ----|
```

- The bucket path acts as a namespace prefix
- The suffix is the sub-key within that namespace
- The separator byte is `/` (0x2F)

### Key Type

```go
type Key struct {
    bucket *Bucket
    suffix []byte
}
```

- `Bytes()` — returns the full concatenated key: `bucket.Path() + suffix`
- `Bucket()` — returns the parent `*Bucket`
- `Suffix()` — returns the suffix portion

### Bucket Type

```go
type Bucket struct {
    path []byte
}
```

- `MakeBucket(path []byte)` — creates a new bucket; ensures trailing `/` separator
- `Bucket(bucketBytes []byte)` — creates a sub-bucket within the current bucket
- `Key(suffix []byte)` — creates a `*Key` within the current bucket
- `Path()` — returns the full bucket path

### Bucket Hierarchy

Buckets support nesting via `Bucket()`:

```go
root := database.MakeBucket([]byte("consensus"))
blocksBucket := root.Bucket([]byte("blocks"))       // "consensus/blocks/"
// blocksBucket.Key(hashBytes)                      // "consensus/blocks/<hash>"
```

---

## Bucket Namespaces

The consensus database is organized into named buckets, each prefixed by the database prefix (configured at consensus factory creation). The full list of buckets defined in the codebase:

### Consensus State Store (`consensusstatestore`)

| Bucket/Key | Name | Description |
|------------|------|-------------|
| **Bucket** | `virtual-utxo-set` | The current virtual UTXO set |
| **Key** | `tips` | Serialized list of current DAG tips |
| **Key** | `importing-pruning-point-utxo-set` | Flag indicating pruning point UTXO import in progress |

### Block Store (`blockstore`)

| Bucket/Key | Name | Description |
|------------|------|-------------|
| **Bucket** | `blocks` | Full block data |
| **Key** | `blocks-count` | Block count counter |

### Block Header Store (`blockheaderstore`)

| Bucket/Key | Name | Description |
|------------|------|-------------|
| **Bucket** | `block-headers` | Block headers |
| **Key** | `block-headers-count` | Header count counter |

### Block Relation Store (`blockrelationstore`)

| Name | Description |
|------|-------------|
| `block-relations` | Block parent/child relationships |

### Block Status Store (`blockstatusstore`)

| Name | Description |
|------|-------------|
| `block-statuses` | Block validation status |

### GHOSTDAG Data Store (`ghostdagdatastore`)

| Name | Description |
|------|-------------|
| `block-ghostdag-data` | GHOSTDAG data for blocks |
| `block-with-trusted-data-ghostdag-data` | GHOSTDAG data for blocks with trusted DAA data |

### UTXO Diff Store (`utxodiffstore`)

| Name | Description |
|------|-------------|
| `utxo-diffs` | Per-block UTXO diffs |
| `utxo-diff-children` | Child pointers for UTXO diffs |

### Reachability Data Store (`reachabilitydatastore`)

| Name | Description |
|------|-------------|
| `reachability-data` | DAG reachability tree data |
| `reachability-reindex-root` | Reindex root hash (single key) |

### Acceptance Data Store (`acceptancedatastore`)

| Name | Description |
|------|-------------|
| `acceptance-data` | Block acceptance data |

### Multiset Store (`multisetstore`)

| Name | Description |
|------|-------------|
| `multisets` | Hash multiset data per block |

### Merge Depth Root Store (`mergedepthrootstore`)

| Name | Description |
|------|-------------|
| `merge-depth-roots` | Finality merge depth roots |

### Finality Store (`finalitystore`)

| Name | Description |
|------|-------------|
| `finality-points` | Finality point data |

### DAA Window Store (`daawindowstore`)

| Name | Description |
|------|-------------|
| `daa-window` | DAA window block references |

### Header Selected Tip Store (`headersselectedtipstore`)

| Name | Description |
|------|-------------|
| `headers-selected-tip` | Current header selected tip hash (single key) |

### Pruning Store (`pruningstore`)

| Name | Description |
|------|-------------|
| `pruning-block-index` | Current pruning point block index (single key) |
| `candidate-pruning-point-hash` | Candidate pruning point hash (single key) |
| `pruning-point-utxo-set` | Pruning point UTXO set snapshot |
| `updating-pruning-point-utxo-set` | Flag for in-progress pruning point update |
| `pruning-point-by-index` | Pruning points keyed by index |
| `last-pruning-time` | Timestamp of last pruning operation |
| `imported-pruning-point-utxos` | Imported pruning point UTXOs during IBD |
| `imported-pruning-point-multiset` | Imported pruning point multiset |

---

## Data Accessor (Domain Layer)

**Package:** `github.com/Eiyaro/Eiyaro/domain/consensus/model`

The domain layer defines its own set of interfaces that mirror the infrastructure layer but with domain-specific types:

### Model Interface Hierarchy

```go
type DBReader interface {
    Get(key DBKey) ([]byte, error)
    Has(key DBKey) (bool, error)
    Cursor(bucket DBBucket) (DBCursor, error)
}

type DBWriter interface {
    DBReader
    Put(key DBKey, value []byte) error
    Delete(key DBKey) error
}

type DBTransaction interface {
    DBWriter
    Rollback() error
    Commit() error
    RollbackUnlessClosed() error
}

type DBManager interface {
    DBWriter
    Begin() (DBTransaction, error)
}
```

### Domain Types

| Type | Description |
|------|-------------|
| `DBKey` | Interface with `Bytes()`, `Bucket()`, `Suffix()` |
| `DBBucket` | Interface with `Bucket()`, `Key()`, `Path()` |
| `DBCursor` | Interface matching `database.Cursor` with domain types |
| `DBReader` | Read-only access to the database |
| `DBWriter` | Read-write access to the database |
| `DBTransaction` | Transactional write access |
| `DBManager` | Full database access including transaction management |

### Differences from Infrastructure Layer

The domain interfaces (`model.DBKey`, `model.DBBucket`, `model.DBCursor`) use domain-specific types rather than raw `database.*` types. This decouples the consensus domain from the infrastructure layer. The conversion functions `dbKeyToDatabaseKey()` and `dbBucketToDatabaseBucket()` in [dbmanager.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/database/dbmanager.go) handle the type translation.

---

## Database Manager

**Package:** `github.com/Eiyaro/Eiyaro/domain/consensus/database` in [dbmanager.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/database/dbmanager.go)

### Construction

```go
func New(db database.Database) model.DBManager
```

`New()` wraps any `database.Database` implementation into a `model.DBManager`. This is the entry point from the consensus factory.

### Retry Mechanism for Get

The `dbManager.Get()` method implements an exponential backoff retry mechanism specifically for handling **empty data responses** (not errors). This addresses a race condition where data written by a commit may not be immediately visible to readers:

- **Max attempts:** 3
- **Initial delay:** 10 ms
- **Max delay:** 500 ms (exponential backoff with cap)

```go
const (
    maxGetRetryAttempts = 3
    initialRetryDelay   = 10 * time.Millisecond
    maxRetryDelay       = 500 * time.Millisecond
)
```

Only empty data triggers retry. Actual errors are returned immediately.

### Migration Support

The `dbManager` itself does not contain migration logic. Database version management and migration are handled at the consensus factory level through the `dbPrefix` and store initialization patterns. Each store initializes its own count keys (e.g., `blocks-count`, `block-headers-count`) on first access.

---

## Bucket Organization

Buckets act as **logical namespaces** within the flat key-value store. They are implemented as byte-array prefixes that are prepended to all keys within that bucket.

### Key Space Layout

```
[prefix]/blocks/                  - Full block data
[prefix]/block-headers/           - Block headers
[prefix]/block-relations/         - Block relationships
[prefix]/block-statuses/          - Block validation status
[prefix]/block-ghostdag-data/     - GHOSTDAG data
[prefix]/acceptance-data/         - Acceptance data
[prefix]/multisets/               - Multiset data
[prefix]/merge-depth-roots/       - Finality data
[prefix]/finality-points/         - Finality points
[prefix]/daa-window/              - DAA window data
[prefix]/utxo-diffs/              - Per-block UTXO diffs
[prefix]/utxo-diff-children/      - UTXO diff children
[prefix]/reachability-data/       - Reachability data
[prefix]/virtual-utxo-set/        - Virtual UTXO set
[prefix]/pruning-point-utxo-set/  - Pruning point UTXO set
[prefix]/pruning-point-by-index/  - Pruning points by index
[prefix]/imported-pruning-point-utxos/ - Imported pruning point UTXOs
[prefix]/tips                     - Tips (single key)
[prefix]/headers-selected-tip     - Header selected tip (single key)
[prefix]/blocks-count             - Block count (single key)
[prefix]/block-headers-count      - Header count (single key)
[prefix]/importing-pruning-point-utxo-set - Import flag (single key)
```

The `[prefix]` is the database prefix byte sequence configured at consensus factory initialization.

### Cursor Scoping

When a cursor is opened on a bucket, it is scoped to that bucket's prefix using the backend's native prefix iteration:

- **LevelDB:** `util.BytesPrefix(bucket.Path())` with `ldb.NewIterator()`
- **Pebble:** Custom `BytesPrefix()` function that computes an upper bound by incrementing the last byte of the prefix

This ensures that `Next()`, `First()`, and `Seek()` only traverse keys within the bucket's namespace. The returned keys from `Key()` have the bucket prefix stripped automatically.

---

## Serialization

**Package:** `github.com/Eiyaro/Eiyaro/domain/consensus/database/serialization`

All structured data is serialized using **Protocol Buffers (protobuf) binary encoding** via the `MarshalVT()`/`UnmarshalVT()` methods. The serialization package provides conversion functions between domain model types and protobuf-generated types.

### Block Serialization

[block.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/database/serialization/block.go) defines bidirectional conversion:

```go
func DomainBlockToDbBlock(domainBlock *externalapi.DomainBlock) *DbBlock
func DbBlockToDomainBlock(dbBlock *DbBlock) (*externalapi.DomainBlock, error)
```

A `DbBlock` contains:
- `Header` — of type `*DbBlockHeader`
- `Transactions` — slice of `*DbTransaction`

Each transaction contains inputs, outputs, and associated data, all serialized via protobuf types.

### Block Header Serialization

[blockheader.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/database/serialization/blockheader.go) defines the header conversion:

```go
func DomainBlockHeaderToDbBlockHeader(domainBlockHeader externalapi.BlockHeader) *DbBlockHeader
func DbBlockHeaderToDomainBlockHeader(dbBlockHeader *DbBlockHeader) (externalapi.BlockHeader, error)
```

The `DbBlockHeader` protobuf message stores:
- `Version` (uint32, validated ≤ MaxUint16)
- `Parents` — block parent hashes
- `HashMerkleRoot` — transaction merkle root
- `AcceptedIDMerkleRoot` — accepted ID merkle root
- `UtxoCommitment` — UTXO commitment hash
- `TimeInMilliseconds` — block timestamp
- `Bits` — difficulty bits
- `Nonce` — proof-of-work nonce
- `DaaScore` — DAA score
- `BlueScore` — blue score
- `BlueWork` — accumulated blue work (stored as bytes, reconstructed as `*big.Int`)
- `PruningPoint` — pruning point hash

### UTXO Entry Serialization

[utxo_entry.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/database/serialization/utxo_entry.go) defines:

```go
func UTXOEntryToDBUTXOEntry(utxoEntry externalapi.UTXOEntry) *DbUtxoEntry
func DBUTXOEntryToUTXOEntry(dbUtxoEntry *DbUtxoEntry) (externalapi.UTXOEntry, error)
```

The `DbUtxoEntry` protobuf message stores:
- `Amount` — satoshi amount
- `ScriptPublicKey` — containing `Script` (bytes) and `Version` (uint32, validated ≤ MaxUint16)
- `BlockDaaScore` — DAA score of the block that created this UTXO
- `IsCoinbase` — coinbase flag

### Script Public Key

```go
func ScriptPublicKeyToDBScriptPublicKey(scriptPublicKey *externalapi.ScriptPublicKey) *DbScriptPublicKey
func DBScriptPublicKeyToScriptPublicKey(dbScriptPublicKey *DbScriptPublicKey) (*externalapi.ScriptPublicKey, error)
```

### Outpoint Serialization

UTXO set keys use serialized outpoints. The outpoint consists of the transaction ID and output index:

```go
func serializeOutpoint(outpoint *externalapi.DomainOutpoint) ([]byte, error)
func deserializeOutpoint(outpointBytes []byte) (*externalapi.DomainOutpoint, error)
```

These use `DomainOutpointToDbOutpoint()` / `DbOutpointToDomainOutpoint()` with protobuf marshal/unmarshal.

### Wire Format

All serialized data uses protobuf's compact binary encoding, providing:
- Efficient storage (smaller than JSON/BSON)
- Backward/forward compatibility via protobuf schema evolution
- Fast encode/decode via generated code

---

## Performance Tuning

### LevelDB Configuration

Configured via environment variables and the `Options()` function in [ldb/options.go](file:///d:/Eiyaro/Eiyaro/infrastructure/db/database/ldb/options.go):

| Parameter | Default | Env Variable | Description |
|-----------|---------|--------------|-------------|
| `Compression` | Snappy | — | Snappy balances speed and storage efficiency |
| `NoSync` | false | — | Ensures data integrity for blockchain data |
| `WriteBuffer` | 64 MiB | — | Larger buffer for bursty writes |
| `BlockCacheCapacity` | 1024 MiB | — | Larger cache for frequent reads |
| `BloomFilter` | 16 bits/key | `HTND_BLOOM_FILTER_LEVEL` / `BLOOM_FILTER_LEVEL` | Reduces disk seeks for point lookups |
| `OpenFilesCacheCapacity` | 1024 | — | Higher file handle cache for concurrent access |
| `CompactionTableSize` | 32 MiB | — | Larger SST files reduce compaction frequency |
| `CompactionTotalSize` | 1024 MiB | — | Larger total before compaction triggers |
| `CompactionL0Trigger` | 8 | `KSDB_COMPACTION_L0_TRIGGER` | Start compaction at more L0 tables |
| `WriteL0SlowdownTrigger` | 24 | `KSDB_WRITE_L0_SLOWDOWN` | Start slowing down later |
| `WriteL0PauseTrigger` | 48 | `KSDB_WRITE_L0_PAUSE` | Hard pause threshold |

### PebbleDB Configuration

Configured via environment variables and `Options()` in [pebble/options.go](file:///d:/Eiyaro/Eiyaro/infrastructure/db/database/pebble/options.go):

| Parameter | Default | Env Variable | Description |
|-----------|---------|--------------|-------------|
| **Cache** | 4096 MiB | `HTND_PEBBLE_CACHE_MB` | Block cache size; higher = better hit rate |
| **MemTableSize** | 512 MiB | `HTND_MEMTABLE_SIZE_MB` | Memtable size per instance |
| **MemTableStopWritesThreshold** | 8 | `HTND_MEMTABLE_STOP_THRESHOLD` | Immutable memtables before stall |
| **Base File Size** | 64 MiB | `HTND_BASE_FILE_SIZE_MB` | Target SST file size at base level |
| **L0 Compaction Threshold** | 64 | `HTND_L0_COMPACTION_THRESHOLD` | L0 files before compaction starts |
| **L0 Stop Writes Threshold** | 200 | `HTND_L0_STOP_WRITES_THRESHOLD` | L0 files before hard stop |
| **L0 Compaction File Threshold** | 64 | `HTND_L0_COMPACTION_FILE_THRESHOLD` | Aligned with compaction trigger |
| **Max Open Files** | 1024 | `HTND_PEBBLE_MAX_OPEN_FILES` | File descriptor limit |
| **L0 Compaction Concurrency** | 12 | `HTND_L0_COMPACTION_CONCURRENCY` | Concurrent L0 compaction workers |
| **Bloom Filter** | 16 bits/key | `HTND_BLOOM_FILTER_LEVEL` | Range 8-20 bits/key |
| **Event Logging** | disabled | `HTND_PEBBLE_LOG_EVENTS` | Enable detailed compaction/flush/stall logging |
| **Event Log Min Duration** | 250 ms | `HTND_PEBBLE_LOG_EVENTS_MIN_MS` | Minimum event duration to log |

### Per-Level Compression Strategy (Pebble)

| Levels | Compression | Block Size | Index Block Size | Bloom Filter |
|--------|-------------|------------|-------------------|--------------|
| L0-L3 | None | 16 KiB | 128 KiB | Yes (16 bits/key) |
| L4-L5 | Snappy | 16 KiB | 128 KiB | Yes (16 bits/key) |
| L6 | Snappy | 16 KiB | 256 KiB | Yes (16 bits/key) |

Levels 0-3 use no compression for fastest access to hot data; higher levels use Snappy for storage efficiency.

### Target File Sizes (Pebble)

| Level | Size Multiplier | Typical Size |
|-------|-----------------|-------------|
| L0 | base × 1 | 64 MiB |
| L1 | base × 4 | 256 MiB |
| L2 | base × 10 | 640 MiB |
| L3 | base × 25 | 1.6 GiB |
| L4 | base × 50 | 3.2 GiB |
| L5 | base × 100 | 6.4 GiB |
| L6 | base × 200 | 12.8 GiB |

### Compaction Tuning

The compaction settings are tuned for high-ingest blockchain workloads:

- **LevelDB:** L0 pause trigger raised to 48 (from default ~12) to reduce write stalls during heavy sequential ingestion
- **Pebble:** L0 thresholds massively increased (64/200 vs default ~4/12) to handle the high block rate before triggering compactions
- **Value separation (Pebble):** Blobs > 1 KiB are separated into blob files to reduce write amplification; this is particularly effective for UTXO entries with script public keys
- **Read-triggered compaction (Pebble):** 256 MiB read compaction rate with 2× sampling multiplier helps compact hot data accessed during IBD

### Value Separation Policy (Pebble)

| Parameter | Value | Description |
|-----------|-------|-------------|
| `Enabled` | true | Enables value separation |
| `MinimumSize` | 1024 bytes | Minimum value size for separation |
| `MaxBlobReferenceDepth` | 100 | Maximum reference chain depth |
| `RewriteMinimumAge` | 24 hours | Minimum blob age before rewriting |
| `TargetGarbageRatio` | 0.20 | Garbage ratio target for blob rewriting |

---

## Environment Variables Summary

### LevelDB (LDB)

| Variable | Description |
|----------|-------------|
| `HTND_BLOOM_FILTER_LEVEL` | Bloom filter bits/key (default: 16) |
| `BLOOM_FILTER_LEVEL` | Fallback bloom filter setting |
| `KSDB_COMPACTION_L0_TRIGGER` | L0 compaction trigger (default: 8) |
| `KSDB_WRITE_L0_SLOWDOWN` | Write L0 slowdown trigger (default: 24) |
| `KSDB_WRITE_L0_PAUSE` | Write L0 pause trigger (default: 48) |

### PebbleDB

| Variable | Description | Default |
|----------|-------------|---------|
| `HTND_BLOOM_FILTER_LEVEL` | Bloom filter bits/key (8-20) | 16 |
| `HTND_MEMTABLE_SIZE_MB` | Memtable size in MiB (>16) | 512 |
| `HTND_MEMTABLE_STOP_THRESHOLD` | Immutable memtable count before stall (≥2) | 8 |
| `HTND_BASE_FILE_SIZE_MB` | Base SST file size in MiB (≥16) | 64 |
| `HTND_PEBBLE_CACHE_MB` | Block cache size in MiB (>256) | 4096 |
| `HTND_L0_COMPACTION_THRESHOLD` | L0 compaction trigger | 64 |
| `HTND_L0_STOP_WRITES_THRESHOLD` | L0 hard stop threshold | 200 |
| `HTND_L0_COMPACTION_FILE_THRESHOLD` | L0 file compaction threshold | 64 |
| `HTND_PEBBLE_MAX_OPEN_FILES` | Max open file descriptors | 1024 |
| `HTND_L0_COMPACTION_CONCURRENCY` | L0 concurrent compaction workers | 12 |
| `HTND_PEBBLE_LOG_EVENTS` | Enable event logging (1/true/yes/y/on) | off |
| `HTND_PEBBLE_LOG_EVENTS_MIN_MS` | Min event duration to log (ms) | 250 |

---

## Error Handling

**Package:** `github.com/Eiyaro/Eiyaro/infrastructure/db/database` in [errors.go](file:///d:/Eiyaro/Eiyaro/infrastructure/db/database/errors.go)

### Sentinel Error

```go
var ErrNotFound = errors.New("not found")
```

This is the primary sentinel error returned when a key does not exist. Both LevelDB and PebbleDB implementations wrap the backend's native not-found error into `database.ErrNotFound`.

### Error Checking

```go
func IsNotFoundError(err error) bool {
    return errors.Is(err, ErrNotFound)
}
```

Use `IsNotFoundError()` to check if an error is caused by a missing key:

```go
value, err := db.Get(key)
if database.IsNotFoundError(err) {
    // key does not exist
}
```

### Error Wrapping Convention

All backend implementations wrap errors with context using `errors.Wrapf` or `errors.WithStack`:

```go
// LevelDB style
return nil, errors.Wrapf(database.ErrNotFound, "key %s not found", key)

// PebbleDB style
return nil, errors.Wrapf(database.ErrNotFound, "key %s not found", key)
```

This preserves the error chain while adding diagnostically useful context.

---

## Usage Examples

### Opening a Database

```go
import (
    "github.com/Eiyaro/Eiyaro/infrastructure/db/database/ldb"
    "github.com/Eiyaro/Eiyaro/infrastructure/db/database/pebble"
    "github.com/Eiyaro/Eiyaro/domain/consensus/database"
)

// LevelDB
ldbDB, err := ldb.NewLevelDB("/path/to/db", 2048) // 2 GiB cache
if err != nil {
    log.Fatal(err)
}
defer ldbDB.Close()

// PebbleDB
pebbleDB, err := pebble.NewPebbleDB("/path/to/db", 4096) // 4 GiB cache
if err != nil {
    log.Fatal(err)
}
defer pebbleDB.Close()

// Wrap as DBManager for consensus access
dbManager := database.New(ldbDB)       // or database.New(pebbleDB)
```

### Performing a Transaction

```go
func writeBlockHeaders(dbManager model.DBManager, bucket model.DBBucket, headers []externalapi.BlockHeader) error {
    tx, err := dbManager.Begin()
    if err != nil {
        return err
    }
    defer tx.RollbackUnlessClosed()

    for _, header := range headers {
        hash := header.BlockHash()
        dbHeader := serialization.DomainBlockHeaderToDbBlockHeader(header)
        serialized, err := dbHeader.MarshalVT()
        if err != nil {
            return err
        }
        key := bucket.Key(hash.ByteSlice())
        if err := tx.Put(key, serialized); err != nil {
            return err
        }
    }

    return tx.Commit()
}
```

### Iterating with a Cursor

```go
func iterateBucket(dbManager model.DBManager, bucket model.DBBucket) error {
    cursor, err := dbManager.Cursor(bucket)
    if err != nil {
        return err
    }
    defer cursor.Close()

    for ok := cursor.First(); ok; ok = cursor.Next() {
        key, err := cursor.Key()
        if err != nil {
            return err
        }
        value, err := cursor.Value()
        if err != nil {
            return err
        }
        // key.Bytes()  -> the full key with bucket prefix
        // key.Suffix() -> the key portion after the bucket prefix
        // value        -> raw protobuf-serialized data
    }

    return nil
}
```

### Seeking to a Specific Key

```go
func seekToOutpoint(dbManager model.DBManager, bucket model.DBBucket, outpoint *externalapi.DomainOutpoint) (externalapi.UTXOEntry, error) {
    cursor, err := dbManager.Cursor(bucket)
    if err != nil {
        return nil, err
    }
    defer cursor.Close()

    serializedOutpoint, err := serialization.DomainOutpointToDbOutpoint(outpoint).MarshalVT()
    if err != nil {
        return nil, err
    }

    seekKey := bucket.Key(serializedOutpoint)
    if err := cursor.Seek(seekKey); err != nil {
        return nil, err
    }

    value, err := cursor.Value()
    if err != nil {
        return nil, err
    }

    dbEntry := &serialization.DbUtxoEntry{}
    if err := dbEntry.UnmarshalVT(value); err != nil {
        return nil, err
    }

    return serialization.DBUTXOEntryToUTXOEntry(dbEntry)
}
```

### Batch Operations

```go
func batchInsert(db database.Database, entries map[*database.Key][]byte) error {
    return db.BatchPut(entries)
}

// With DBManager
func batchInsertDomain(dbManager model.DBManager, entries map[model.DBKey][]byte) error {
    for key, value := range entries {
        if err := dbManager.Put(key, value); err != nil {
            return err
        }
    }
    return nil
}
```

### Using Bucket Hierarchy

```go
func createStoreBuckets(prefixBucket model.DBBucket) struct {
    blocks      model.DBBucket
    headers     model.DBBucket
    utxo        model.DBBucket
} {
    return struct {
        blocks  model.DBBucket
        headers model.DBBucket
        utxo    model.DBBucket
    }{
        blocks:  prefixBucket.Bucket([]byte("blocks")),
        headers: prefixBucket.Bucket([]byte("block-headers")),
        utxo:    prefixBucket.Bucket([]byte("virtual-utxo-set")),
    }
}
```

### Checking Key Existence

```go
func ensureKeyExists(dbManager model.DBManager, key model.DBKey) error {
    exists, err := dbManager.Has(key)
    if err != nil {
        return err
    }
    if !exists {
        return errors.New("key does not exist")
    }
    return nil
}
```

### Safe Deletion

```go
// Delete is idempotent — no error if key doesn't exist
func safeDelete(dbManager model.DBManager, key model.DBKey) error {
    return dbManager.Delete(key)
}
```

---

## Compaction

Both backends expose a `Compact()` method that triggers full-range compaction:

```go
// LevelDB
func (db *LevelDB) Compact() error {
    return db.ldb.CompactRange(util.Range{Start: nil, Limit: nil})
}

// PebbleDB
func (db *DB) Compact() error {
    return db.db.Compact(context.Background(), nil, []byte{0xff, 0xff, 0xff, 0xff}, false)
}
```

Compaction consolidates SST files, reclaims space from tombstones, and improves read performance. It is typically triggered manually or via administrative tooling (e.g., `ldbtool`).

---

## Implementation Notes

### LevelDB Transaction Read Semantics

`LevelDBTransaction.Get()` and `LevelDBTransaction.Has()` read directly from the **underlying database**, not from the transaction's batch. This means:

- Data written by `Put()` within the same transaction is **not visible** to subsequent `Get()` calls in that transaction
- Data written by other concurrent transactions **may be visible** (no snapshot isolation)

```go
// LevelDBTransaction.Get implementation
func (tx *LevelDBTransaction) Get(key *database.Key) ([]byte, error) {
    // Bypasses the batch entirely, reads from db
    return tx.db.Get(key)
}
```

### Pebble Transaction Read Semantics

`DBTransaction.Get()` uses an **indexed batch** that supports reading uncommitted modifications:

```go
func (tx *DBTransaction) Get(key *database.Key) ([]byte, error) {
    // Check batch modifications first
    if exists, ok := tx.keyModifications[string(key.Bytes())]; ok {
        if !exists {
            return nil, errors.Wrapf(database.ErrNotFound, ...)
        }
        // Read from indexed batch
        data, closer, err := tx.batch.Get(key.Bytes())
        // ...
    }
    // Fall back to the database
    return tx.db.Get(key)
}
```

This is a significant behavioral difference between the two backends. Code should **not** rely on intra-transaction read visibility.

### Pebble Get Data Safety

Pebble's `db.Get()` returns data that is **unsafe after closing the returned closer**. The Pebble implementation explicitly clones the data before closing:

```go
data, closer, err := db.db.Get(key.Bytes())
// ...
valueCopy := bytes.Clone(data)
closer.Close()
return valueCopy, nil
```

### Cursor Lifecycle Management

**LevelDB cursors** are not tracked by the database. Callers must explicitly `Close()` every cursor.

**Pebble cursors** are tracked via `registerCursor()`/`deregisterCursor()`. On `DB.Close()`, all remaining open cursors are automatically closed. This prevents resource leaks but adds internal bookkeeping overhead.

---

## Testing

Each backend package contains test files:

- `ldb/leveldb_test.go` — LevelDB integration tests
- `ldb/transaction_test.go` — LevelDB transaction tests
- `ldb/cursor_test.go` — LevelDB cursor tests
- `ldb/fuse_test.go` — LevelDB filesystem tests
- `pebble/pebble_test.go` — Pebble integration tests
- `pebble/transaction_test.go` — Pebble transaction tests
- `pebble/cursor_test.go` — Pebble cursor tests

Additionally, the consensus data store layer has test utilities in `datastructures/testutils/testutils.go` that create test database instances using `consensusdatabase.MakeBucket([]byte("datastructures-test"))`.

---

## Dependencies

### LevelDB Backend

- `github.com/syndtr/goleveldb/leveldb` — core LevelDB implementation
- `github.com/syndtr/goleveldb/leveldb/opt` — options configuration
- `github.com/syndtr/goleveldb/leveldb/util` — range and prefix utilities
- `github.com/syndtr/goleveldb/leveldb/errors` — corruption detection
- `github.com/syndtr/goleveldb/leveldb/filter` — Bloom filter support

### PebbleDB Backend

- `github.com/cockroachdb/pebble/v2` — core Pebble implementation
- `github.com/cockroachdb/pebble/v2/bloom` — Bloom filter policy
- `github.com/cockroachdb/pebble/v2/sstable` — SST compression profiles

### Common

- `github.com/pkg/errors` — error wrapping and stack traces
- `github.com/Eiyaro/Eiyaro/infrastructure/logger` — structured logging
- Protocol Buffers + VT (vtprotobuf) — data serialization

---

## Notes & Caveats

### Transaction Read Semantics (LevelDB vs PebbleDB)

There is a critical behavioral difference between the two backends regarding intra-transaction read visibility. **LevelDB transactions** read directly from the underlying database, meaning data written by `Put()` within the same transaction is **not visible** to subsequent `Get()` calls in that transaction. **Pebble transactions** use an indexed batch that supports reading uncommitted modifications — `Get()` within a Pebble transaction can see data written earlier in the same transaction. Code should **never** rely on intra-transaction read visibility; always structure database updates to avoid reading your own uncommitted writes.

### Cursor Lifecycle

**LevelDB cursors** are not tracked by the database; callers must explicitly `Close()` every cursor or resources leak permanently. **Pebble cursors** are registered with the parent DB and automatically closed when the DB shuts down, but you should still `Close()` them explicitly in normal code paths. All cursor methods panic after `Close()` — always use `defer cursor.Close()` immediately after creation.

### Corruption Recovery Differences

LevelDB attempts automatic recovery of corrupted files via `leveldb.RecoverFile()` on open, preserving existing data where possible. PebbleDB takes a nuclear approach: if `pebble.ErrCorruption` is detected on open, the entire database directory is removed and a fresh one created. This means a corrupted PebbleDB node must fully resync from the network, while a corrupted LevelDB node may recover partially.

### Migration from LevelDB to PebbleDB

There is no in-place migration tool. Switching backends requires a full resync of the blockchain from the network. Plan for significant downtime if migrating a large database.

### Disk Space

The blockchain database grows continuously during operation. Archival nodes that set `--archival` retain all historical data and can grow to hundreds of gigabytes. Non-archival nodes prune old data at each pruning point update, keeping the database at a relatively stable size.

---

## FAQ

### Should I use LevelDB or PebbleDB?

PebbleDB is recommended for most deployments due to better write performance, configurable per-level compression, value separation for large blobs, and cursor lifecycle management. LevelDB remains a stable, battle-tested fallback. Use PebbleDB unless you have a specific reason to prefer LevelDB.

### How do I migrate from LevelDB to PebbleDB?

There is no in-place migration. Set the `--dbtype` flag (or environment variables) to switch backends, then perform a full blockchain resync. Start the node with the new backend pointing to a fresh data directory and let it sync from the network.

### What happens if the database gets corrupted?

LevelDB attempts automatic recovery on open via `leveldb.RecoverFile()`. PebbleDB removes the entire corrupted database directory and creates a fresh one. In either case, the node will resync missing data from the P2P network. Corrupted database = data loss and resync; always ensure you have good peer connectivity.

### How much disk space does the blockchain database need?

A non-archival mainnet node typically requires 50–150 GB depending on network age and pruning settings. Archival nodes (`--archival`) retain all historical block data and UTXO diffs and can grow to several hundred gigabytes. PebbleDB with value separation may reduce disk usage compared to LevelDB.

### Can I change database performance settings after initial sync?

Yes. Most PebbleDB settings (cache size, memtable size, compaction thresholds) are controlled via environment variables like `HTND_PEBBLE_CACHE_MB` and can be adjusted between restarts. LevelDB tuning is mostly compile-time via the `Options()` function. Restart the node for changes to take effect.