package processor

import (
	"bitbucket.org/free5gc-team/nef/internal/consumer"
	nefctx "bitbucket.org/free5gc-team/nef/internal/context"
	"bitbucket.org/free5gc-team/nef/internal/notifier"
	"bitbucket.org/free5gc-team/nef/pkg/factory"
)

type nef interface {
	Context() *nefctx.NefContext
	Config() *factory.Config
	Consumer() *consumer.Consumer
	Notifier() *notifier.Notifier
}

type Processor struct {
	nef
}

type HandlerResponse struct {
	Status  int
	Headers map[string][]string
	Body    interface{}
}

func NewProcessor(nef nef) (*Processor, error) {
	handler := &Processor{
		nef: nef,
	}

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
