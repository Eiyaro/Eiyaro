package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Eiyaro/Eiyaro/infrastructure/config"
	"github.com/Eiyaro/Eiyaro/infrastructure/db/database"
	"github.com/Eiyaro/Eiyaro/infrastructure/db/database/ldb"
	"github.com/Eiyaro/Eiyaro/infrastructure/db/database/pebble"
	"github.com/Eiyaro/Eiyaro/infrastructure/logger"
	"github.com/Eiyaro/Eiyaro/infrastructure/os/execenv"
	"github.com/Eiyaro/Eiyaro/infrastructure/os/limits"
	"github.com/Eiyaro/Eiyaro/infrastructure/os/signal"
	"github.com/Eiyaro/Eiyaro/infrastructure/os/winservice"
	"github.com/Eiyaro/Eiyaro/util/panics"
	"github.com/Eiyaro/Eiyaro/util/profiling"
	"github.com/Eiyaro/Eiyaro/version"
)

const (
	leveldbCacheSizeMiB  = 256
	pebbledbCacheSizeMiB = 2048
	defaultDataDirname   = "datadir2"
)

var desiredLimits = &limits.DesiredLimits{
	FileLimitWant: 2048,
	FileLimitMin:  1024,
}

var serviceDescription = &winservice.ServiceDescription{
	Name:        "eyarodsvc",
	DisplayName: "eyarod Service",
	Description: "Downloads and stays synchronized with the Eiyaro blockDAG and " +
		"provides DAG services to applications.",
}

type eyarodApp struct {
	cfg *config.Config
}

// Start starts the Eiyaro node app, and blocks until it finishes running
func Start() error {
	execenv.Initialize(desiredLimits)

	// Load configuration and parse command line. This function also
	// initializes logging and configures it accordingly.
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	defer logger.BackendLog.Close()
	defer panics.HandlePanic(log, "MAIN", nil)

	app := &eyarodApp{cfg: cfg}
	// Call serviceMain on Windows to handle running as a service. When
	// the return isService flag is true, exit now since we ran as a
	// service. Otherwise, just fall through to normal operation.
	if runtime.GOOS == "windows" {
		isService, err := winservice.WinServiceMain(app.main, serviceDescription, cfg)
		if err != nil {
			return err
		}
		if isService {
			return nil
		}
	}

	return app.main(nil)
}

func (app *eyarodApp) main(startedChan chan<- struct{}) error {
	// Get a channel that will be closed when a shutdown signal has been
	// triggered either from an OS signal such as SIGINT (Ctrl+C) or from
	// another subsystem such as the RPC server.
	interrupt := signal.InterruptListener()
	defer log.Info("Shutdown complete")

	// Show version at startup.
	log.Infof("Version %s", version.Version())

	// Enable http profiling server if requested.
	if app.cfg.Profile != "" {
		profiling.Start(app.cfg.Profile, log)
	}
	profiling.TrackHeap(app.cfg.AppDir, log)

	// Return now if an interrupt signal was triggered.
	if signal.InterruptRequested(interrupt) {
		return nil
	}

	if app.cfg.ResetDatabase {
		err := removeDatabase(app.cfg)
		if err != nil {
			log.Error(err)
			return err
		}
	}

	// Open the database
	databaseContext, err := openDB(app.cfg)
	if err != nil {
		log.Errorf("Loading database failed: %+v", err)
		return err
	}

	defer func() {
		log.Infof("Gracefully shutting down the database...")
		err := databaseContext.Close()
		if err != nil {
			log.Errorf("Failed to close the database: %s", err)
		}
	}()

	// Return now if an interrupt signal was triggered.
	if signal.InterruptRequested(interrupt) {
		return nil
	}

	// Create componentManager and start it.
	componentManager, err := NewComponentManager(app.cfg, databaseContext, interrupt)
	if err != nil {
		log.Errorf("Unable to start eyarod: %+v", err)
		return err
	}

	defer func() {
		log.Infof("Gracefully shutting down eyarod...")

		shutdownDone := make(chan struct{})
		go func() {
			componentManager.Stop()
			shutdownDone <- struct{}{}
		}()

		const shutdownTimeout = 2 * time.Minute

		select {
		case <-shutdownDone:
		case <-time.After(shutdownTimeout):
			log.Criticalf("Graceful shutdown timed out %s. Terminating...", shutdownTimeout)
		}
		log.Infof("eyarod shutdown complete")
	}()

	componentManager.Start()

	if startedChan != nil {
		startedChan <- struct{}{}
	}

	// Wait until the interrupt signal is received from an OS signal or
	// shutdown is requested through one of the subsystems such as the RPC
	// server.
	<-interrupt
	return nil
}

// dbPath returns the path to the block database given a database type.
func databasePath(cfg *config.Config) string {
	return filepath.Join(cfg.AppDir, defaultDataDirname)
}

func removeDatabase(cfg *config.Config) error {
	dbPath := databasePath(cfg)
	return os.RemoveAll(dbPath)
}

func openDB(cfg *config.Config) (database.Database, error) {
	dbPath := databasePath(cfg)

	err := checkDatabaseVersion(dbPath)
	if err != nil {
		return nil, err
	}

	if strings.EqualFold(cfg.DbType, "leveldb") {
		log.Infof("Loading LevelDB database  from '%s'", dbPath)
		db, err := ldb.NewLevelDB(dbPath, leveldbCacheSizeMiB)
		if err != nil {
			return nil, err
		}
		return db, nil
	}
	// For pebble and all other db types, use PebbleDB
	log.Infof("Loading %s database from '%s'", cfg.DbType, dbPath)
	db, err := pebble.NewPebbleDB(dbPath, pebbledbCacheSizeMiB)
	if err != nil {
		return nil, err
	}
	return db, nil
}
