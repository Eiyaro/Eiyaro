package nogopow

import (
	"math/big"
	"os"
	"sync/atomic"
	"time"
)

const (
	ModeNormal PowMode = iota
	ModeFake
	ModeTest
)

type PowMode uint32

type Config struct {
	PowMode        PowMode
	CacheDir       string
	Log            Logger
	ConsensusParams *ConsensusParams
}

type ConsensusParams struct {
	TargetBlockTime time.Duration
	MinDifficulty   *big.Int
	MaxDifficulty   *big.Int
	PIControllerEnabled bool
	DifficultyAdjustWindow int
}

type Logger interface {
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}

type defaultLogger struct{}

func (l *defaultLogger) Info(msg string, args ...interface{}) {
	println("[INFO] " + msg)
}

func (l *defaultLogger) Warn(msg string, args ...interface{}) {
	println("[WARN] " + msg)
}

func (l *defaultLogger) Error(msg string, args ...interface{}) {
	println("[ERROR] " + msg)
}

func (l *defaultLogger) Debug(msg string, args ...interface{}) {
	println("[DEBUG] " + msg)
}

func DefaultConfig() Config {
	cacheDir := os.Getenv("NOGOPOW_CACHE_DIR")
	if cacheDir == "" {
		cacheDir = "./nogopow_cache"
	}

	return Config{
		PowMode:  ModeNormal,
		CacheDir: cacheDir,
		Log:      &defaultLogger{},
		ConsensusParams: &ConsensusParams{
			TargetBlockTime:      20 * time.Second,
			MinDifficulty:        big.NewInt(1000000),
			MaxDifficulty:        new(big.Int).Lsh(big.NewInt(1), 256),
			PIControllerEnabled:  true,
			DifficultyAdjustWindow: 100,
		},
	}
}

func (c *Config) IsFake() bool {
	return atomic.LoadUint32((*uint32)(&c.PowMode)) != uint32(ModeNormal)
}

func (c *Config) IsTest() bool {
	return atomic.LoadUint32((*uint32)(&c.PowMode)) == uint32(ModeTest)
}
