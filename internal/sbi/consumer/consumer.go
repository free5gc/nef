package consumer

import (
	"github.com/free5gc/nef/internal/logger"
	"github.com/free5gc/nef/pkg/app"
	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/nrf/NFDisc"
	"github.com/free5gc/openapi/nrf/NFMgmt"
	"github.com/free5gc/openapi/pcf/PolAuth"
	"github.com/free5gc/openapi/udr/DR"
	"github.com/free5gc/util/nfheartbeat"
)

type nef interface {
	app.App
}

type Consumer struct {
	nef

	// consumer services
	*nnrfService
	*npcfService
	*nudrService
}

func NewConsumer(nef nef) (*Consumer, error) {
	c := &Consumer{
		nef: nef,
	}

	c.nnrfService = &nnrfService{
		consumer:        c,
		nfDiscClients:   make(map[string]*NFDisc.APIClient),
		nfMngmntClients: make(map[string]*NFMgmt.APIClient),
	}
	heartbeat, err := nfheartbeat.NewRunner(
		nrfRegistrar{c.nnrfService},
		func() int32 { return c.Config().GetNfHeartBeatTimer() },
		logger.ConsumerLog,
	)
	if err != nil {
		return nil, err
	}
	c.heartbeat = heartbeat

	c.npcfService = &npcfService{
		consumer: c,
		clients:  make(map[string]*PolAuth.APIClient),
	}

	c.nudrService = &nudrService{
		consumer: c,
		clients:  make(map[string]*DR.APIClient),
	}
	return c, nil
}

func handleAPIServiceNoResponse(err error) (int, interface{}) {
	detail := "server no response"
	if err != nil {
		detail = err.Error()
	}
	logger.ConsumerLog.Errorf("APIService error: %s", detail)
	pd := openapi.ProblemDetailsSystemFailure(detail)
	return int(pd.Status), pd
}
