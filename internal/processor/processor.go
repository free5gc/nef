package processor

import (
	"bitbucket.org/free5gc-team/nef/internal/consumer"
	nefctx "bitbucket.org/free5gc-team/nef/internal/context"
	"bitbucket.org/free5gc-team/nef/internal/factory"
	"bitbucket.org/free5gc-team/nef/internal/notifier"
)

type Processor struct {
	cfg      *factory.Config
	nefCtx   *nefctx.NefContext
	consumer *consumer.Consumer
	notifier *notifier.Notifier
}

type HandlerResponse struct {
	Status  int
	Headers map[string][]string
	Body    interface{}
}

func NewProcessor(cfg *factory.Config, nefCtx *nefctx.NefContext, consumer *consumer.Consumer,
	notifier *notifier.Notifier) *Processor {

	handler := &Processor{cfg: cfg, nefCtx: nefCtx, consumer: consumer, notifier: notifier}

	return handler
}
