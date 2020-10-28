package processor

import (
	"bitbucket.org/free5gc-team/nef/internal/context"
)

type Processor struct {
	nefCtx *context.NefContext
}

type HandlerResponse struct {
	Status  int
	Headers map[string][]string
	Body    interface{}
}

func NewProcessor(nefCtx *context.NefContext) *Processor {
	handler := &Processor{nefCtx: nefCtx}
	handler.init()

	return handler
}

func (h *Processor) init() {
}
