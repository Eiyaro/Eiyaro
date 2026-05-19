package handshake

import (
	"github.com/Eiyaro/Eiyaro/infrastructure/logger"
	"github.com/Eiyaro/Eiyaro/util/panics"
)

var (
	log   = logger.RegisterSubSystem("PROT")
	spawn = panics.GoroutineWrapperFunc(log)
)
