package main

import (
	"github.com/Eiyaro/Eiyaro/infrastructure/logger"
	"github.com/Eiyaro/Eiyaro/util/panics"
)

var (
	backendLog = logger.NewBackend()
	log        = backendLog.Logger("JSTT")
	spawn      = panics.GoroutineWrapperFunc(log)
)
