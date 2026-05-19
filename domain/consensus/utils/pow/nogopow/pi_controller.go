package nogopow

import (
	"math"
	"math/big"
	"sync"
	"time"
)

const (
	defaultKp           = 0.15
	defaultKi           = 0.03
	integralDecay       = 0.97
	integralClampMin    = -3.0
	integralClampMax    = 3.0
	minDifficultyFactor = 0.5
	maxDifficultyFactor = 2.0
)

type PIDGains struct {
	Kp float64
	Ki float64
}

type DifficultyAdjuster struct {
	mutex              sync.RWMutex
	consensusParams    *ConsensusParams
	integralAccumulator float64
	gains              PIDGains
	windowSize         int
	blockTimes         []time.Duration
}

func NewDifficultyAdjuster(params *ConsensusParams) *DifficultyAdjuster {
	if params == nil {
		defaultParams := DefaultConfig().ConsensusParams
		params = defaultParams
	}
	
	windowSize := params.DifficultyAdjustWindow
	if windowSize <= 0 {
		windowSize = 100
	}
	
	return &DifficultyAdjuster{
		consensusParams:     params,
		integralAccumulator: 0.0,
		gains: PIDGains{
			Kp: defaultKp,
			Ki: defaultKi,
		},
		windowSize: windowSize,
		blockTimes: make([]time.Duration, 0, windowSize),
	}
}

func (da *DifficultyAdjuster) CalcDifficulty(chain ChainHeaderReader, time uint64, parent *Header) *big.Int {
	da.mutex.Lock()
	defer da.mutex.Unlock()
	
	if parent == nil || parent.Difficulty == nil {
		return da.consensusParams.MinDifficulty
	}
	
	if len(da.blockTimes) < da.windowSize {
		return da.calculateInitialDifficulty(parent)
	}
	
	return da.calculatePIDifficultyLocked(parent)
}

func (da *DifficultyAdjuster) calculateInitialDifficulty(parent *Header) *big.Int {
	if parent.Time == 0 {
		return da.consensusParams.MinDifficulty
	}
	
	diff := new(big.Int).Set(parent.Difficulty)
	
	if diff.Cmp(da.consensusParams.MinDifficulty) < 0 {
		return new(big.Int).Set(da.consensusParams.MinDifficulty)
	}
	
	return da.enforceBoundaryConditionsLocked(diff)
}

func (da *DifficultyAdjuster) calculatePIDifficultyLocked(parent *Header) *big.Int {
	targetTime := da.consensusParams.TargetBlockTime
	
	avgTime := da.calculateAverageBlockTime()
	if avgTime <= 0 {
		avgTime = targetTime
	}
	
	errorTerm := float64(targetTime-avgTime) / float64(targetTime)
	
	proportional := da.gains.Kp * errorTerm
	
	da.integralAccumulator += errorTerm * da.gains.Ki
	da.integralAccumulator *= integralDecay
	
	if da.integralAccumulator < integralClampMin {
		da.integralAccumulator = integralClampMin
	} else if da.integralAccumulator > integralClampMax {
		da.integralAccumulator = integralClampMax
	}
	
	integral := da.integralAccumulator
	
	totalAdjustment := 1.0 + proportional + integral
	
	baseDifficulty := new(big.Int).Set(parent.Difficulty)
	
	adjustmentFactor := big.NewFloat(float64(totalAdjustment))
	baseDiffFloat := new(big.Float).SetInt(baseDifficulty)
	
	newDiffFloat := new(big.Float).Mul(baseDiffFloat, adjustmentFactor)
	
	newDifficulty, _ := newDiffFloat.Int(nil)
	
	newDifficulty = da.enforceBoundaryConditionsLocked(newDifficulty)
	
	da.blockTimes = da.blockTimes[1:]
	
	return newDifficulty
}

func (da *DifficultyAdjuster) calculateAverageBlockTime() time.Duration {
	if len(da.blockTimes) == 0 {
		return 0
	}
	
	total := int64(0)
	for _, t := range da.blockTimes {
		total += int64(t)
	}
	
	avg := total / int64(len(da.blockTimes))
	return time.Duration(avg)
}

func (da *DifficultyAdjuster) enforceBoundaryConditionsLocked(difficulty *big.Int) *big.Int {
	if difficulty == nil {
		return da.consensusParams.MinDifficulty
	}
	
	minDiff := new(big.Int).Set(da.consensusParams.MinDifficulty)
	maxDiff := new(big.Int).Set(da.consensusParams.MaxDifficulty)
	
	if difficulty.Cmp(minDiff) < 0 {
		return new(big.Int).Set(minDiff)
	}
	
	if difficulty.Cmp(maxDiff) > 0 {
		return new(big.Int).Set(maxDiff)
	}
	
	return difficulty
}

func (da *DifficultyAdjuster) UpdateBlockTime(actualTime time.Duration) {
	da.mutex.Lock()
	defer da.mutex.Unlock()
	
	da.blockTimes = append(da.blockTimes, actualTime)
	
	if len(da.blockTimes) > da.windowSize {
		da.blockTimes = da.blockTimes[1:]
	}
}

func (da *DifficultyAdjuster) Reset() {
	da.mutex.Lock()
	defer da.mutex.Unlock()
	
	da.integralAccumulator = 0.0
	da.blockTimes = da.blockTimes[:0]
}

func (da *DifficultyAdjuster) GetStats() (avgBlockTime time.Duration, integral float64) {
	da.mutex.RLock()
	defer da.mutex.RUnlock()
	
	return da.calculateAverageBlockTime(), da.integralAccumulator
}

func clampFloat64(val, min, max float64) float64 {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

func absFloat64(val float64) float64 {
	return math.Abs(val)
}
