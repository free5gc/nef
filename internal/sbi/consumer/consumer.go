package consumer

import (
	"github.com/free5gc/nef/internal/logger"
	"github.com/free5gc/nef/pkg/app"
	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/amf/EvtExpos"
	"github.com/free5gc/openapi/nrf/NFDisc"
	"github.com/free5gc/openapi/nrf/NFMgmt"
	"github.com/free5gc/openapi/pcf/PolAuth"
	"github.com/free5gc/openapi/udm/SDM"
	"github.com/free5gc/openapi/udr/DR"
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
	*namfService
	*nudmService
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

	c.npcfService = &npcfService{
		consumer: c,
		clients:  make(map[string]*PolAuth.APIClient),
	}

	c.nudrService = &nudrService{
		consumer: c,
		clients:  make(map[string]*DR.APIClient),
	}

	c.namfService = &namfService{
		consumer: c,
		clients:  make(map[string]*EvtExpos.APIClient),
	}

	c.nudmService = &nudmService{
		consumer: c,
		clients:  make(map[string]*SDM.APIClient),
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
