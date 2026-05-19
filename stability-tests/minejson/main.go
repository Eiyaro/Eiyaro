package main

import (
	"github.com/Eiyaro/Eiyaro/domain/consensus"
	"github.com/Eiyaro/Eiyaro/stability-tests/common"
	"github.com/Eiyaro/Eiyaro/stability-tests/common/mine"
	"github.com/Eiyaro/Eiyaro/stability-tests/common/rpc"
	"github.com/Eiyaro/Eiyaro/util/panics"
	"github.com/Eiyaro/Eiyaro/util/profiling"
	"github.com/pkg/errors"
)

func main() {
	defer panics.HandlePanic(log, "minejson-main", nil)
	err := parseConfig()
	if err != nil {
		panic(errors.Wrap(err, "error parsing configuration"))
	}
	defer backendLog.Close()
	common.UseLogger(backendLog, log.Level())

	cfg := activeConfig()
	if cfg.Profile != "" {
		profiling.Start(cfg.Profile, log)
	}
	rpcClient, err := rpc.ConnectToRPC(&cfg.Config, cfg.NetParams())
	if err != nil {
		panic(errors.Wrap(err, "error connecting to JSON-RPC server"))
	}
	defer func() { _ = rpcClient.Disconnect() }()

	dataDir, err := common.TempDir("minejson")
	if err != nil {
		panic(err)
	}

	consensusConfig := consensus.Config{Params: *cfg.NetParams()}

	err = mine.FromFile(cfg.DAGFile, &consensusConfig, rpcClient, dataDir)
	if err != nil {
		panic(errors.Wrap(err, "error in mine.FromFile"))
	}
}
