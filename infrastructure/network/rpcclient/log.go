package rpcclient

import (
	"github.com/Eiyaro/Eiyaro/infrastructure/logger"
	"github.com/Eiyaro/Eiyaro/util/panics"
)

var (
	log   = logger.RegisterSubSystem("RPCC")
	spawn = panics.GoroutineWrapperFunc(log)
)
