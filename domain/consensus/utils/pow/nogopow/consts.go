package nogopow

import "time"

// Matrix computation constants
const (
	MatrixSize       = 256               // Matrix dimension: 256x256
	MatrixCount      = 128               // Number of matrices
	FixedPointShift  = 30                // Fixed-point shift bits (2^30 precision)
	FixedPointFactor = 1 << FixedPointShift // Fixed-point scaling factor
	MatrixModulo     = 1000              // Matrix element modulo for value clamping
)

// PI Controller constants
const (
	DefaultKp            = 0.15              // Proportional gain coefficient
	DefaultKi            = 0.03              // Integral gain coefficient
	IntegralDecay        = 0.97              // Integral accumulator decay factor
	IntegralClampMin     = -3.0              // Integral lower bound
	IntegralClampMax     = 3.0               // Integral upper bound
	DifficultyAdjustMin  = 0.5               // Minimum difficulty adjustment factor (50%)
	DifficultyAdjustMax  = 2.0               // Maximum difficulty adjustment factor (200%)
)

// Cache constants
const (
	MaxCacheItems        = 64                // Maximum items in LRU cache
	CacheCleanupInterval = 5 * time.Minute   // Cache cleanup interval
)

// Timing constants
const (
	TargetBlockTime      = 20 * time.Second  // Target time between blocks
	DifficultyWindow     = 100               // Number of blocks for difficulty adjustment
)
