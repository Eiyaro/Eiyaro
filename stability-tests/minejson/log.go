package main

import (
	"github.com/Eiyaro/Eiyaro/infrastructure/logger"
)

var (
	backendLog = logger.NewBackend()
	log        = backendLog.Logger("MNJS")
)
