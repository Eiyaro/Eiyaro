package integration

import (
	"os"
	"testing"

	"github.com/Eiyaro/Eiyaro/infrastructure/logger"
)

func TestMain(m *testing.M) {
	logLevel := os.Getenv("EIYARO_TEST_LOGLEVEL")
	if logLevel == "" {
		logLevel = "error"
	}
	_ = logger.ParseAndSetLogLevels(logLevel)
	level, ok := logger.LevelFromString(logLevel)
	if !ok {
		level = logger.LevelError
	}
	logger.InitLogStdout(level)

	os.Exit(m.Run())
}
