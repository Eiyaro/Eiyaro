package rpchandlers

import (
	"github.com/Eiyaro/Eiyaro/infrastructure/logger"
	"github.com/Eiyaro/Eiyaro/util/panics"
)

var (
	log   = logger.RegisterSubSystem("RPCS")
	spawn = panics.GoroutineWrapperFunc(log)
)
