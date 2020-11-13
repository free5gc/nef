package processor

import (
	"bitbucket.org/free5gc-team/nef/internal/consumer"
	"bitbucket.org/free5gc-team/nef/internal/context"
	"bitbucket.org/free5gc-team/nef/internal/factory"
)

type Processor struct {
	cfg      *factory.Config
	nefCtx   *context.NefContext
	consumer *consumer.Consumer
}

type HandlerResponse struct {
	Status  int
	Headers map[string][]string
	Body    interface{}
}

func NewProcessor(nefCfg *factory.Config, nefCtx *context.NefContext, consumer *consumer.Consumer) *Processor {
	handler := &Processor{cfg: nefCfg, nefCtx: nefCtx, consumer: consumer}

	return handler
}
