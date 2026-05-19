package consensus

import (
	"github.com/Eiyaro/Eiyaro/infrastructure/logger"
	"github.com/Eiyaro/Eiyaro/util/panics"
)

var (
	log   = logger.RegisterSubSystem("BDAG")
	spawn = panics.GoroutineWrapperFunc(log)
)
