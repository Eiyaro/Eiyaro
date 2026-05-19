package connmanager

import (
	"github.com/Eiyaro/Eiyaro/infrastructure/logger"
	"github.com/Eiyaro/Eiyaro/util/panics"
)

var (
	log   = logger.RegisterSubSystem("CMGR")
	spawn = panics.GoroutineWrapperFunc(log)
)
