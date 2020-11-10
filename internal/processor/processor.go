package processor

import (
	"bitbucket.org/free5gc-team/nef/internal/context"
	"bitbucket.org/free5gc-team/nef/internal/factory"
)

type Processor struct {
	cfg    *factory.Config
	nefCtx *context.NefContext
}

type HandlerResponse struct {
	Status  int
	Headers map[string][]string
	Body    interface{}
}

func NewProcessor(nefCfg *factory.Config, nefCtx *context.NefContext) *Processor {
	handler := &Processor{cfg: nefCfg, nefCtx: nefCtx}

	return handler
}
