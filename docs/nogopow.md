# NogoPow Proof-of-Work Algorithm

## Overview

NogoPow is an ASIC-resistant Proof-of-Work algorithm designed for the Eiyaro blockchain. It uses 256×256 fixed-point matrix multiplication combined with Blake3 hashing. The design makes it memory-hard and computationally intensive, favoring CPU miners and resisting specialized hardware acceleration.

The algorithm is implemented in two cooperating layers:

| Layer | Package | Purpose |
|-------|---------|---------|
| Engine | `nogopow` | Standalone mining engine with config, caching, difficulty adjustment, and a `sync.Pool`-backed dense matrix implementation |
| Consensus Integration | `pow` | Wires NogoPow into the Eiyaro consensus model using `externalapi` types, Xoshiro256++ RNG, and the `State` struct |

---

## Design Philosophy

- **ASIC Resistance**: Matrix operations require large memory bandwidth (256×256 int64 elements = 512 KiB per matrix), not just raw compute throughput. ASICs optimized for bitwise hashing cannot trivially accelerate dense linear algebra.
- **CPU Friendly**: Optimized for general-purpose CPUs with multi-threading support (default 4 threads). Uses cache-friendly blocking and Kahan summation for numerical stability.
- **Verification Efficiency**: Verification is fast — a single `SealHash` call compared against the difficulty target. Mining requires the full matrix multiplication pipeline.
- **Deterministic**: Given the same block header, the same matrices are always generated. The Xoshiro256++ PRNG is seeded from the block header hash, guaranteeing reproducibility.

---

## Algorithm Overview

The PoW computation follows these steps:

1.  **Generate matrices** — A 256×256 matrix `A` is generated from the block header using Xoshiro256++ RNG (seed = block header hash). A second 256×256 matrix `B` is generated from the block number.
2.  **Matrix multiplication** — `C = A × B` is computed using fixed-point arithmetic with multi-threaded blocking.
3.  **Blake3 hash** — The result matrix `C` is serialized and hashed with Blake3, combined with the seed bytes.
4.  **Target comparison** — The resulting hash is compared against the difficulty target. A valid block satisfies `hash < target`.

```
┌──────────────────┐     ┌──────────────────┐
│   Block Header   │     │   Block Number   │
└────────┬─────────┘     └────────┬─────────┘
         │                        │
         ▼                        ▼
┌──────────────────┐     ┌──────────────────┐
│ Xoshiro256++ RNG │     │  BlockNumber     │
│ (seeded by hash) │     │  → Matrix B      │
│ → Matrix A       │     │  (256×256 int64) │
│ (256×256 fixed)  │     │                  │
└────────┬─────────┘     └────────┬─────────┘
         │                        │
         └────────┬───────────────┘
                  │
                  ▼
         ┌──────────────────┐
         │  C = A × B       │
         │  (4 threads,     │
         │   blocked mul)   │
         └────────┬─────────┘
                  │
                  ▼
         ┌──────────────────┐
         │  Blake3(C, seed) │
         └────────┬─────────┘
                  │
                  ▼
         ┌──────────────────┐
         │  hash < target?  │
         └──────────────────┘
```

---

## Core Components

### 1. Matrix Generation

#### Matrix A — From Block Header (Xoshiro256++)

Defined in [pow.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/pow/pow.go) and [xoshiro.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/pow/xoshiro.go):

```go
func generateNogopowMatrix(hash *externalapi.DomainHash) *nogopowMatrix
```

- The block header hash (32 bytes) seeds a **Xoshiro256++** PRNG.
- Each `Uint64()` call produces 64 random bits. Each 4-bit nibble is converted to a fixed-point `int64` value via `int16ToFixed` (left-shift by `FixedPointShift = 30`).
- 16 elements are packed per `Uint64()` call (j steps by 16), producing a deterministic 256×256 matrix from 4096 random draws.

**Xoshiro256++ state** ([xoshiro.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/pow/xoshiro.go)):

```go
type xoShiRo256PlusPlus struct {
    s0, s1, s2, s3 uint64  // 256-bit state, seeded from 32-byte hash
}
```

The state transition mixes all four 64-bit registers with rotations and XORs, producing high-quality pseudorandom output suitable for deterministic PoW.

#### Matrix B — From Block Number

Defined in [pow.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/pow/pow.go):

```go
func newBlockNumberMatrix(blockNumber uint64) *nogopowMatrix
```

Each element `B[i][j]` is computed as:

```
B[i][j] = (blockNumber × (i + j + 1)) mod MatrixModulo
```

Where `MatrixModulo = 1000`. The result is then converted to fixed-point:

```go
func int64ToFixed(val int64) int64 {
    return val << nogopowFixedPointShift  // val << 30
}
```

This ensures that different block numbers produce distinct matrices, preventing precomputation attacks.

#### Engine-Level Matrix Generation

In the standalone engine ([engine.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/pow/nogopow/engine.go)):

```go
func (e *NogopowEngine) computePoW(header *Header, target *big.Int) BlockNonce
```

This uses the `SeedHash` (Blake3 of the header) as a direct seed for generating matrix elements rather than Xoshiro256++. The two approaches are logically equivalent — both produce deterministic matrices from header data.

### 2. Fixed-Point Arithmetic

Defined in [consts.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/pow/nogopow/consts.go):

| Constant | Value | Description |
|----------|-------|-------------|
| `FixedPointShift` | `30` | Number of bits for fractional part |
| `FixedPointFactor` | `1073741824` (2^30) | Scaling factor for float ↔ fixed conversion |
| `MatrixModulo` | `1000` | Modulo for element clamping |

Fixed-point representation stores real numbers as scaled integers: `fixed_value = real_value × 2^30`. After multiplication, the result is right-shifted by 30 bits to rescale:

```go
product := av * bv           // Multiply fixed-point values
product >>= FixedPointShift   // Rescale: divide by 2^30
sum += product                // Accumulate
```

This avoids floating-point non-determinism across different CPU architectures and compilers.

#### Conversion Functions ([matrix.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/pow/nogopow/matrix.go))

```go
func toFixed(val float64) int64 {
    return int64(val * float64(FixedPointFactor))
}

func fromFixed(val int64) float64 {
    return float64(val) / float64(FixedPointFactor)
}

func toFixedShift(val int64) int64   // Safe left-shift with overflow protection
```

### 3. Matrix Multiplication

#### Dense Matrix ([matrix.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/pow/nogopow/matrix.go))

```go
type denseMatrix struct {
    data []int64   // Flat slice of rows × cols elements
    rows int
    cols int
}
```

A `sync.Pool` (`matrixPool`) manages matrix allocation to reduce GC pressure during mining.

#### Multi-Threaded Pooled Multiplication

```go
func mulMatrixPooled(a, b *denseMatrix, numThreads int) *denseMatrix
```

- Distributes rows across `numThreads` goroutines (default 4).
- Each goroutine computes its assigned rows independently — no locks needed.
- Uses `sync.WaitGroup` for synchronization.

#### Blocked Multiplication (Cache-Optimized)

```go
func mulMatrixBlocked(a, b *denseMatrix, blockSize int) *denseMatrix
```

Standard O(n³) triple loop with blocking (tiling) to improve cache locality. Iterates over sub-blocks of the matrices to keep working sets within L1/L2 cache sizes.

#### Safe Arithmetic Wrappers ([safe_math.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/pow/nogopow/safe_math.go))

All multiplication and addition in the matrix pipeline uses overflow-checked wrappers:

```go
func SafeMulInt64(a, b int64) (int64, error)   // Checks a > MaxInt64/b before multiplying
func SafeAddInt64(a, b int64) (int64, error)   // Checks a > MaxInt64-b before adding
func SafeSubInt64(a, b int64) (int64, error)   // Checks underflow conditions
func SafeDivInt64(a, b int64) (int64, error)   // Checks division by zero and MinInt64/-1
func SafeLeftShift(val int64, shift uint) (int64, error)  // Checks shift overflow
func SafeRightShift(val int64, shift uint) (int64, error) // Validates shift amount
```

Overflow conditions are handled by clamping to `math.MaxInt64` / `math.MinInt64`, ensuring the algorithm never panics.

#### Kahan Summation

```go
func kahanSum(values []float64) float64
func kahanSumInt64(values []int64) int64
```

Implements the Kahan summation algorithm to reduce accumulated floating-point error, ensuring cross-platform consistency in hash results.

### 4. Blake3 Hashing

NogoPow uses **Blake3** (not Blake2b) as its hash function. Blake3 is significantly faster than Blake2b and SHA-256 on modern CPUs while maintaining 256-bit security.

#### SealHash ([hash.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/pow/nogopow/hash.go))

```go
func SealHash(header *Header) Hash
```

Serializes the block header (parent hash, coinbase, state root, tx hash, number, gas limit, timestamp, extra data, nonce, difficulty) via a deterministic RLP-like encoding, then hashes with Blake3. This is the hash used for both mining and verification.

#### Matrix Hashing ([matrix.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/pow/nogopow/matrix.go))

```go
func hashMatrix(m *denseMatrix, seed []byte) Hash
```

- Writes seed bytes to the Blake3 hasher.
- Serializes all 256×256 = 65,536 `int64` matrix elements as little-endian bytes.
- Produces a 32-byte output hash.

#### Consensus-Integrated Hashing ([pow.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/pow/pow.go))

```go
func nogopowHashMatrix(mat *nogopowMatrix, seed []byte) *externalapi.DomainHash
```

Equivalent operation using `externalapi.DomainHash` types for consensus integration.

### 5. Difficulty Target

#### Target Calculation ([hash.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/pow/nogopow/hash.go))

```go
func difficultyToTarget(difficulty *big.Int) *big.Int
```

The relationship between difficulty and target:

```
maxTarget = 2^256 - 1
target = maxTarget / difficulty
```

A block is valid when:

```
Blake3(header_serialized) < target
```

Higher difficulty → smaller target → harder to mine. The inverse conversion is:

```go
func targetToDifficulty(target *big.Int) *big.Int   // difficulty = maxTarget / target
```

#### Hash Validation

```go
func verifyHashAgainstTarget(hash Hash, target *big.Int) bool {
    return new(big.Int).SetBytes(hash[:]).Cmp(target) <= 0
}
```

### 6. PI Controller (Difficulty Adjustment)

Defined in [pi_controller.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/pow/nogopow/pi_controller.go):

The `DifficultyAdjuster` implements a **Proportional-Integral (PI) controller** for dynamic difficulty adjustment.

#### Constants

| Parameter | Value | Description |
|-----------|-------|-------------|
| `defaultKp` | `0.15` | Proportional gain — responds to instantaneous error |
| `defaultKi` | `0.03` | Integral gain — corrects accumulated bias |
| `integralDecay` | `0.97` | Decay factor to prevent integral windup |
| `integralClampMin` | `-3.0` | Integral accumulator lower bound |
| `integralClampMax` | `3.0` | Integral accumulator upper bound |
| `minDifficultyFactor` | `0.5` | Minimum adjustment factor (difficulty won't drop below 50%) |
| `maxDifficultyFactor` | `2.0` | Maximum adjustment factor (difficulty won't rise above 200%) |

#### Algorithm

```
error = (targetBlockTime - avgBlockTime) / targetBlockTime
proportionalTerm = Kp × error
integralTerm += Ki × error
integralTerm *= integralDecay        // slowly forget old errors
integralTerm = clamp(integralTerm, -3.0, 3.0)
adjustmentFactor = 1.0 + proportionalTerm + integralTerm
newDifficulty = parentDifficulty × adjustmentFactor
newDifficulty = clamp(newDifficulty, minDifficulty, maxDifficulty)
```

#### Key Properties

- **Smooth adjustment**: PI control prevents oscillation unlike simple proportional controllers.
- **Integral anti-windup**: Decay factor prevents the integral term from accumulating unbounded error.
- **Bounded output**: Difficulty changes are clamped to [50%, 200%] of the parent difficulty, preventing extreme swings.
- **Window-based**: Maintains a sliding window of recent block times (default 100 blocks).

#### Struct

```go
type DifficultyAdjuster struct {
    mutex              sync.RWMutex
    consensusParams    *ConsensusParams
    integralAccumulator float64
    gains              PIDGains
    windowSize         int
    blockTimes         []time.Duration
}
```

#### Methods

```go
func NewDifficultyAdjuster(params *ConsensusParams) *DifficultyAdjuster
func (da *DifficultyAdjuster) CalcDifficulty(chain ChainHeaderReader, time uint64, parent *Header) *big.Int
func (da *DifficultyAdjuster) UpdateBlockTime(actualTime time.Duration)
func (da *DifficultyAdjuster) Reset()
func (da *DifficultyAdjuster) GetStats() (avgBlockTime time.Duration, integral float64)
```

### 7. Cache System

Defined in [cache.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/pow/nogopow/cache.go):

A thread-safe LRU cache backed by `container/list` stores up to 64 computed PoW results, keyed by `(headerHash, nonce)`. Uses `singleflight.Group` to prevent duplicate computation (thundering herd) when multiple goroutines request the same uncached key.

```go
type Cache struct {
    lruCache *simpleLRU          // Max 64 items
    lock     sync.RWMutex
    config   *Config
    group    singleflight.Group  // Deduplication
    memPool  sync.Pool           // Buffer reuse
}
```

---

## Configuration

### Config Struct ([config.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/pow/nogopow/config.go))

```go
type Config struct {
    PowMode         PowMode          // ModeNormal, ModeFake, or ModeTest
    CacheDir        string           // Directory for cache storage
    Log             Logger           // Structured logger interface
    ConsensusParams  *ConsensusParams // Difficulty and timing parameters
}
```

### PowMode

| Mode | Value | Behavior |
|------|-------|----------|
| `ModeNormal` | `0` | Full PoW computation |
| `ModeFake` | `1` | Returns fake nonce (for testing) |
| `ModeTest` | `2` | Test mode (configurable behavior) |

### ConsensusParams

```go
type ConsensusParams struct {
    TargetBlockTime        time.Duration  // Default: 20s
    MinDifficulty          *big.Int       // Default: 1,000,000
    MaxDifficulty          *big.Int       // Default: 2^256
    PIControllerEnabled    bool           // Default: true
    DifficultyAdjustWindow int            // Default: 100 blocks
}
```

### Default Configuration

```go
func DefaultConfig() Config {
    return Config{
        PowMode:  ModeNormal,
        CacheDir: os.Getenv("NOGOPOW_CACHE_DIR") or "./nogopow_cache",
        Log:      &defaultLogger{},
        ConsensusParams: &ConsensusParams{
            TargetBlockTime:        20 * time.Second,
            MinDifficulty:          big.NewInt(1000000),
            MaxDifficulty:          new(big.Int).Lsh(big.NewInt(1), 256),
            PIControllerEnabled:    true,
            DifficultyAdjustWindow: 100,
        },
    }
}
```

### All Constants Summary ([consts.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/pow/nogopow/consts.go))

| Constant | Value | Category |
|----------|-------|----------|
| `MatrixSize` | `256` | Matrix |
| `MatrixCount` | `128` | Matrix |
| `FixedPointShift` | `30` | Fixed-point |
| `FixedPointFactor` | `1073741824` | Fixed-point |
| `MatrixModulo` | `1000` | Matrix |
| `DefaultKp` | `0.15` | PI Controller |
| `DefaultKi` | `0.03` | PI Controller |
| `IntegralDecay` | `0.97` | PI Controller |
| `IntegralClampMin` | `-3.0` | PI Controller |
| `IntegralClampMax` | `3.0` | PI Controller |
| `DifficultyAdjustMin` | `0.5` | PI Controller |
| `DifficultyAdjustMax` | `2.0` | PI Controller |
| `MaxCacheItems` | `64` | Cache |
| `CacheCleanupInterval` | `5 min` | Cache |
| `TargetBlockTime` | `20s` | Timing |
| `DifficultyWindow` | `100` | Timing |

---

## Integration with Consensus

### State Struct ([pow.go / nogopow_heavy.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/pow/pow.go))

```go
type State struct {
    nogopowMatA  *nogopowMatrix       // Matrix A: from header hash via Xoshiro256++
    nogopowMatB  *nogopowMatrix       // Matrix B: from block number
    Timestamp    int64                 // Block timestamp in milliseconds
    Nonce        uint64                // Mining nonce
    Target       big.Int               // Difficulty target
    PrevHeader   externalapi.DomainHash // Previous block header hash
    BlockVersion uint16                // Block version
}
```

### State Creation

```go
func NewState(header externalapi.MutableBlockHeader) *State
```

1. Extracts `target` from header `Bits` field (compact difficulty representation).
2. Temporarily zeroes timestamp and nonce to compute the "pure" previous header hash.
3. Generates matrix A via `generateNogopowMatrix(prevHeader)` using Xoshiro256++.
4. Generates matrix B via `newBlockNumberMatrix(header.DAAScore())`.

### PoW Value Computation

```go
func (state *State) CalculateProofOfWorkValue() (*big.Int, *externalapi.DomainHash)
```

1. Writes prev header bytes, timestamp, zeros, and nonce to a Blake3 hash writer to produce `powHash`.
2. Multiplies matrices: `C = A × B` (4 threads, fixed-point).
3. Hashes: `resultHash = Blake3(C, powHash)`.
4. Returns `(BigInt(resultHash), resultHash)`.

### Nonce Increment

```go
func (state *State) IncrementNonce() {
    state.Nonce++
}
```

Simple increment — matrix A and B are precomputed and reused across nonce iterations. Only the `powHash` (which embeds the nonce) changes between attempts.

### PoW Verification

```go
func (state *State) CheckProofOfWork(block *externalapi.DomainBlock, powSkip bool) bool
```

- Computes the expected PoW value from the header.
- If `powSkip` is true, only checks `powValue < target`.
- Otherwise, also verifies the submitted `PoWHash` matches the computed hash (protection against block withholding).

```go
func CheckProofOfWorkByBits(header externalapi.MutableBlockHeader, block *externalapi.DomainBlock, powSkip bool) bool
```

Convenience wrapper that creates a `State` and calls `CheckProofOfWork`.

### Block Level

```go
func BlockLevel(header externalapi.BlockHeader, maxBlockLevel int) int
```

Computes a block's "level" as `max(0, maxBlockLevel - proofOfWorkValue.BitLen())`. Blocks with lower PoW values (more work) get higher levels, useful for chain selection heuristics.

---

## API Reference

### NogopowEngine ([engine.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/pow/nogopow/engine.go))

```go
func New(config *Config, chain ChainReader, sealCh chan<- *Block) *NogopowEngine
```
Creates a new mining engine with the given configuration and chain reader. The `sealCh` channel receives successfully mined blocks.

```go
func NewFaker() *NogopowEngine
```
Creates an engine in `ModeFake` — returns a hardcoded fake nonce for testing.

```go
func (e *NogopowEngine) VerifyHeader(chain ChainHeaderReader, header *Header, sealOnly bool) error
```
Full header verification: validates block time ordering, difficulty calculation, and (optionally) seal validity.

```go
func (e *NogopowEngine) VerifySealOnly(chain ChainHeaderReader, header *Header) error
```
Verifies only the PoW seal (hash vs target). Skips in fake mode.

```go
func (e *NogopowEngine) Seal(ctx context.Context, chain ChainReader, block *Block, sealCh chan<- *Block) error
```
Starts the mining process for a block. Non-blocking — launches a goroutine. Uses `atomic.CompareAndSwapInt32` to prevent concurrent mining.

```go
func (e *NogopowEngine) CalcDifficulty(chain ChainHeaderReader, time uint64, parent *Header) *big.Int
```
Calculates the required difficulty for the next block. Delegates to the PI controller when enabled.

```go
func (e *NogopowEngine) HashRate() uint64
```
Returns the current mining hashrate in hashes/second. Decays to 0 after 5 seconds of inactivity.

```go
func (e *NogopowEngine) Close() error
```
Signals exit to the mining goroutine and waits for cleanup.

### State (consensus integration)

```go
func NewState(header externalapi.MutableBlockHeader) *State
func (state *State) CalculateProofOfWorkValue() (*big.Int, *externalapi.DomainHash)
func (state *State) IncrementNonce()
func (state *State) CheckProofOfWork(block *externalapi.DomainBlock, powSkip bool) bool
func CheckProofOfWorkByBits(header externalapi.MutableBlockHeader, block *externalapi.DomainBlock, powSkip bool) bool
func BlockLevel(header externalapi.BlockHeader, maxBlockLevel int) int
func SetUseHoohashCLibrary(use bool)  // No-op, kept for backward compatibility
```

### Hash Utilities ([hash.go](file:///d:/Eiyaro/Eiyaro/domain/consensus/utils/pow/nogopow/hash.go))

```go
func SealHash(header *Header) Hash
func BytesToHash(b []byte) Hash
func BigToHash(b *big.Int) Hash
func difficultyToTarget(difficulty *big.Int) *big.Int
func targetToDifficulty(target *big.Int) *big.Int
func computeBlake3Hash(data []byte) Hash
func computeSHA3Hash(data []byte) Hash
func verifyHashAgainstTarget(hash Hash, target *big.Int) bool
```

### Error Variables

```go
var ErrInvalidSeal      = errors.New("invalid seal")
var ErrInvalidDifficulty = errors.New("invalid difficulty")
var ErrInvalidTime      = errors.New("invalid time")
var ErrNonceInvalid     = errors.New("nonce invalid")
```

---

## Usage Examples

### Creating PoW State and Computing a Proof-of-Work Value

```go
import (
    "github.com/Eiyaro/Eiyaro/domain/consensus/model/externalapi"
    "github.com/Eiyaro/Eiyaro/domain/consensus/utils/pow"
)

header := &externalapi.DomainMutableBlockHeader{}
header.SetBits(difficulty.BigToCompact(targetBigInt))
header.SetTimeInMilliseconds(currentTimeMs)
header.SetNonce(0)
header.SetDAAScore(blockNumber)

state := pow.NewState(header)

for {
    powValue, powHash := state.CalculateProofOfWorkValue()
    if powValue.Cmp(&state.Target) <= 0 {
        fmt.Printf("Valid PoW found! Nonce: %d, Hash: %x\n", state.Nonce, powHash.ByteSlice())
        break
    }
    state.IncrementNonce()
}
```

### Verifying PoW

```go
state := pow.NewState(header)
isValid := state.CheckProofOfWork(block, false)
if isValid {
    fmt.Println("Block has valid Proof-of-Work")
}
```

### Using the Standalone Engine

```go
cfg := nogopow.DefaultConfig()
engine := nogopow.New(&cfg, chainReader, sealChannel)

err := engine.VerifyHeader(chainHeaderReader, header, false)
if err != nil {
    log.Fatalf("header verification failed: %v", err)
}

err = engine.Seal(context.Background(), chainReader, block, sealChannel)
if err != nil {
    log.Fatalf("sealing failed: %v", err)
}
```

### Checking Engine Hashrate

```go
go func() {
    for {
        time.Sleep(5 * time.Second)
        rate := engine.HashRate()
        fmt.Printf("Hashrate: %d H/s\n", rate)
    }
}()
```

---

## Performance Characteristics

### Memory Requirements

| Component | Size |
|-----------|------|
| Matrix A (256×256 int64) | 512 KiB |
| Matrix B (256×256 int64) | 512 KiB |
| Result Matrix C | 512 KiB |
| **Per-mining total** | **~1.5 MiB** |

With `sync.Pool` backing, matrices are reused across mining iterations. The cache stores up to 64 precomputed results (32 bytes each = 2 KiB).

### Multi-Threading

- Default: 4 threads for matrix multiplication.
- Row-parallel: each thread computes a contiguous block of rows independently.
- Scaling is approximately linear up to the number of physical cores, bound by memory bandwidth for large matrices.

### Cache Optimization

- **Blocked multiplication** (`mulMatrixBlocked`) uses tiling to keep sub-blocks in L1/L2 cache.
- **Matrix pool** (`matrixPool`) eliminates allocation overhead during mining loops.
- **Singleflight** deduplication prevents redundant computation when multiple goroutines request the same (header, nonce) pair.

### Timing

- Target block time: **20 seconds** (configurable via `ConsensusParams.TargetBlockTime`).
- Difficulty adjustment window: **100 blocks**.
- Mining timeout: **2× target block time** (40 seconds default).

---

## Notes & Caveats

### Fixed-Point Precision

- **30-bit fractional precision** means values are accurate to approximately 1 part in 10^9.
- After fixed-point multiplication and right-shift, some low-order bits are lost. This is intentional and deterministic — all nodes compute identically.
- The Kahan summation algorithm mitigates cumulative rounding error in accumulation steps.

### Thread Safety

- The `NogopowEngine` uses `sync.RWMutex` and `atomic.Int32` for concurrent access.
- `DifficultyAdjuster` is protected by `sync.RWMutex`.
- Matrix multiplication workers operate on independent row ranges — no shared mutable state.
- The `Cache` uses separate locks for LRU operations and singleflight deduplication.

### Platform Considerations

- **Pure Go implementation** — no CGo dependencies, no platform-specific assembly.
- `SetUseHoohashCLibrary(use bool)` is a no-op kept for backward compatibility with older Eiyaro code that supported an optional C library.
- Build constraint `//go:build !linux || !arm64` on `nogopow_heavy.go` indicates the pure Go path is the default.

### Relationship to Difficulty Adjustment

- When `PIControllerEnabled` is `false`, difficulty remains at the parent block's value (no adjustment).
- The PI controller requires a full window of block times before it activates; during the warm-up period, the parent difficulty is used directly (clamped to `MinDifficulty` if below).
- The integral accumulator decays by `0.97` each adjustment, preventing long-term bias accumulation.

### Cryptographic Security

- **Blake3** provides 256-bit preimage and collision resistance, suitable for PoW.
- **Xoshiro256++** provides a 256-bit state with a period of 2^256 − 1, ensuring no repeated sequences during mining.
- Matrix elements are generated deterministically from the header — the PoW is non-interactive and publicly verifiable.
- The difficulty-to-target conversion uses `math/big` arbitrary precision to handle 256-bit targets exactly.

### Backward Compatibility

The `SetUseHoohashCLibrary` function and the `nogopow_heavy.go` filename reflect the algorithm's lineage: NogoPow replaces an earlier "Hoohash" algorithm that optionally used a C library. The pure Go NogoPow implementation is now the canonical path.

---

## FAQ

### Is GPU mining possible with NogoPow?

NogoPow is designed to be ASIC-resistant but is not specifically optimized for GPUs. The algorithm uses 256×256 fixed-point matrix multiplication (512 KiB per matrix) which requires significant memory bandwidth. While GPUs can perform matrix math in parallel, the Blake3 hashing step and the deterministic matrix generation from Xoshiro256++ make GPU acceleration less effective than on CPUs. The design explicitly favors general-purpose CPU mining.

### Why does mining use so much CPU?

Each mining attempt performs a full 256×256 matrix multiplication (O(n³) = ~16.7 million operations) followed by a Blake3 hash over 65,536 matrix elements. This is computationally expensive by design — the memory-hard matrix operations prevent trivial acceleration. The default configuration uses 4 threads for parallelized matrix multiplication, and each nonce increment requires recomputing the hash (matrices A and B are precomputed and reused).

### How is the mining difficulty calculated?

Difficulty is managed by a Proportional-Integral (PI) controller with `Kp = 0.15` and `Ki = 0.03`. The controller measures the average block time over a 100-block sliding window, computes the error against the 20-second target, and adjusts difficulty by a factor clamped between 0.5× and 2.0× of the parent difficulty. The relationship is: `target = maxTarget / difficulty`, and a block is valid when `Blake3(header) < target`.

### Can the NogoPow algorithm be changed in the future?

The PoW algorithm is a consensus-critical component, so any change would require a network-wide hard fork or a graceful transition period via BIP0009-style miner voting (95% threshold on mainnet). The `SetUseHoohashCLibrary` function is already a no-op stub, indicating that the codebase supports algorithm evolution. The `PowMode` enum (Normal, Fake, Test) provides testing flexibility without affecting mainnet consensus.

### What happens if the fixed-point arithmetic overflows?

All multiplication and addition in the matrix pipeline use overflow-checked wrappers (`SafeMulInt64`, `SafeAddInt64`, etc.) that clamp to `math.MaxInt64` or `math.MinInt64` instead of wrapping. The Kahan summation algorithm mitigates cumulative rounding error. Since the same overflow behavior is deterministic across all nodes, consensus is preserved — every node computes exactly the same hash from the same header data.