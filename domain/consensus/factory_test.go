package consensus

import (
	"os"
	"testing"

	"github.com/Eiyaro/Eiyaro/domain/prefixmanager/prefix"
	"github.com/Eiyaro/Eiyaro/infrastructure/db/database/ldb"

	"github.com/Eiyaro/Eiyaro/domain/dagconfig"
)

func TestNewConsensus(t *testing.T) {
	f := NewFactory()

	config := &Config{Params: dagconfig.DevnetParams}

	tmpDir, err := os.MkdirTemp("", "TestNewConsensus")
	if err != nil {
		return
	}

	db, err := ldb.NewLevelDB(tmpDir, 8)
	if err != nil {
		t.Fatalf("error in NewLevelDB: %s", err)
	}

	_, shouldMigrate, err := f.NewConsensus(config, db, &prefix.Prefix{}, nil)
	if err != nil {
		t.Fatalf("error in NewConsensus: %+v", err)
	}

	if shouldMigrate {
		t.Fatalf("A fresh consensus should never return shouldMigrate=true")
	}
}
