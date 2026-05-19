package standalone

import (
	"github.com/Eiyaro/Eiyaro/infrastructure/logger"
	"github.com/Eiyaro/Eiyaro/util/panics"
)

var (
	log   = logger.RegisterSubSystem("NTAR")
	spawn = panics.GoroutineWrapperFunc(log)
)
