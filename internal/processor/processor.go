package processor

import (
	"bitbucket.org/free5gc-team/nef/internal/consumer"
	nefctx "bitbucket.org/free5gc-team/nef/internal/context"
	"bitbucket.org/free5gc-team/nef/internal/notifier"
	"bitbucket.org/free5gc-team/nef/pkg/factory"
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

func NewProcessor(nefCtx *nefctx.NefContext, consumer *consumer.Consumer,
	notifier *notifier.Notifier) (*Processor, error) {
	handler := &Processor{cfg: nefCtx.Config(), nefCtx: nefCtx, consumer: consumer, notifier: notifier}

	return handler, nil
}

func addLocationheader(header map[string][]string, location string) {
	locations := header["Location"]
	if locations == nil {
		header["Location"] = []string{location}
	} else {
		header["Location"] = append(locations, location)
	}
}
