package nogopow

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrInvalidSeal      = errors.New("invalid seal")
	ErrInvalidDifficulty = errors.New("invalid difficulty")
	ErrInvalidTime      = errors.New("invalid time")
	ErrNonceInvalid     = errors.New("nonce invalid")
)

type NogopowEngine struct {
	config     *Config
	sealCh     chan<- *Block
	exitCh     chan struct{}
	wg         sync.WaitGroup
	lock       sync.RWMutex
	running    int32
	hashrate   *hashrateTracker
	cache      *Cache
	diffAdjuster *DifficultyAdjuster
}

type hashrateTracker struct {
	rate      uint64
	lastUpdate time.Time
	lock      sync.Mutex
}

func (h *hashrateTracker) Update(hashes int, duration time.Duration) {
	h.lock.Lock()
	defer h.lock.Unlock()
	
	rate := uint64(float64(hashes) / duration.Seconds())
	h.rate = rate
	h.lastUpdate = time.Now()
}

func (h *hashrateTracker) Rate() uint64 {
	h.lock.Lock()
	defer h.lock.Unlock()
	
	if time.Since(h.lastUpdate) > 5*time.Second {
		return 0
	}
	return h.rate
}

func New(config *Config, chain ChainReader, sealCh chan<- *Block) *NogopowEngine {
	if config == nil {
		cfg := DefaultConfig()
		config = &cfg
	}
	
	engine := &NogopowEngine{
		config:     config,
		sealCh:     sealCh,
		exitCh:     make(chan struct{}),
		hashrate:   &hashrateTracker{},
		cache:      NewCache(config),
		diffAdjuster: NewDifficultyAdjuster(config.ConsensusParams),
	}
	
	return engine
}

func NewFaker() *NogopowEngine {
	cfg := DefaultConfig()
	cfg.PowMode = ModeFake
	
	return &NogopowEngine{
		config:   &cfg,
		sealCh:   nil,
		exitCh:   make(chan struct{}),
		hashrate: &hashrateTracker{},
		cache:    NewCache(&cfg),
	}
}

func (e *NogopowEngine) VerifyHeader(chain ChainHeaderReader, header *Header, sealOnly bool) error {
	if header.Number == nil {
		return fmt.Errorf("header number is nil")
	}
	
	if header.Difficulty == nil {
		return fmt.Errorf("header difficulty is nil")
	}
	
	parent := chain.HeaderByNumber(new(big.Int).Sub(header.Number, big.NewInt(1)))
	if parent == nil {
		return fmt.Errorf("parent header not found for number %d", header.Number)
	}
	
	if header.Time <= parent.Time {
		return ErrInvalidTime
	}
	
	expectedDiff := e.CalcDifficulty(chain, header.Time, parent)
	if header.Difficulty.Cmp(expectedDiff) != 0 {
		return fmt.Errorf("invalid difficulty: got %s, expected %s", header.Difficulty.String(), expectedDiff.String())
	}
	
	if sealOnly {
		return e.VerifySealOnly(chain, header)
	}
	
	return nil
}

func (e *NogopowEngine) VerifySealOnly(chain ChainHeaderReader, header *Header) error {
	if e.config.IsFake() {
		return nil
	}
	
	return e.verifySeal(chain, header)
}

func (e *NogopowEngine) verifySeal(chain ChainHeaderReader, header *Header) error {
	if header.Number == nil || header.Difficulty == nil {
		return ErrInvalidSeal
	}
	
	target := difficultyToTarget(header.Difficulty)
	result := SealHash(header)
	
	if new(big.Int).SetBytes(result[:]).Cmp(target) > 0 {
		return ErrNonceInvalid
	}
	
	return nil
}

func (e *NogopowEngine) Seal(ctx context.Context, chain ChainReader, block *Block, sealCh chan<- *Block) error {
	if e.config.IsFake() {
		nonce := BlockNonce{}
		copy(nonce[:], []byte("fake_nonce"))
		block.header.Nonce = nonce
		sealCh <- block
		return nil
	}
	
	if !atomic.CompareAndSwapInt32(&e.running, 0, 1) {
		return fmt.Errorf("engine already running")
	}
	defer atomic.StoreInt32(&e.running, 0)
	
	e.wg.Add(1)
	defer e.wg.Wait()
	
	go e.mineBlock(ctx, chain, block, sealCh)
	
	return nil
}

func (e *NogopowEngine) mineBlock(ctx context.Context, chain ChainReader, block *Block, sealCh chan<- *Block) {
	defer e.wg.Done()
	
	header := block.header
	if header == nil {
		return
	}
	
	startTime := time.Now()
	maxDuration := e.config.ConsensusParams.TargetBlockTime
	
	target := difficultyToTarget(header.Difficulty)
	
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-e.exitCh:
			return
		case <-ticker.C:
			start := time.Now()
			
			nonce := e.computePoW(header, target)
			
			duration := time.Since(start)
			e.hashrate.Update(MatrixCount, duration)
			
			result := SealHash(header)
			if new(big.Int).SetBytes(result[:]).Cmp(target) <= 0 {
				header.Nonce = nonce
				select {
				case sealCh <- block:
				default:
				}
				return
			}
			
			if time.Since(startTime) > maxDuration*2 {
				e.config.Log.Warn("mining timeout", "duration", time.Since(startTime))
				return
			}
		}
	}
}

func (e *NogopowEngine) computePoW(header *Header, target *big.Int) BlockNonce {
	seed := SealHash(header)
	
	matA := GetMatrix()
	defer PutMatrix(matA)
	
	matB := GetMatrix()
	defer PutMatrix(matB)
	
	for i := 0; i < MatrixSize; i++ {
		for j := 0; j < MatrixSize; j++ {
			val := int64(seed[i%32]) * int64(seed[j%32])
			val %= MatrixModulo
			matA.Set(i, j, toFixedShift(val))
		}
	}
	
	for i := 0; i < MatrixSize; i++ {
		for j := 0; j < MatrixSize; j++ {
			val := int64(header.Number.Uint64()) * int64(i+j)
			val %= MatrixModulo
			matB.Set(i, j, toFixedShift(val))
		}
	}
	
	resultMat := mulMatrixPooled(matA, matB, 4)
	if resultMat == nil {
		var zero BlockNonce
		return zero
	}
	
	defer func() {
		for i := range resultMat.data {
			resultMat.data[i] = 0
		}
		matrixPool.Put(resultMat)
	}()
	
	hashResult := hashMatrix(resultMat, seed[:])
	
	var nonce BlockNonce
	copy(nonce[:], hashResult[:])
	
	return nonce
}

func (e *NogopowEngine) CalcDifficulty(chain ChainHeaderReader, time uint64, parent *Header) *big.Int {
	if e.config.ConsensusParams.PIControllerEnabled {
		return e.diffAdjuster.CalcDifficulty(chain, time, parent)
	}
	
	return parent.Difficulty
}

func (e *NogopowEngine) HashRate() uint64 {
	return e.hashrate.Rate()
}

func (e *NogopowEngine) Close() error {
	close(e.exitCh)
	e.wg.Wait()
	return nil
}

func checkSolution(header *Header, target *big.Int) bool {
	result := SealHash(header)
	return new(big.Int).SetBytes(result[:]).Cmp(target) <= 0
}
