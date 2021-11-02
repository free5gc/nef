package consumer

import (
	"net/http"

	nefctx "bitbucket.org/free5gc-team/nef/internal/context"
	"bitbucket.org/free5gc-team/nef/internal/logger"
	"bitbucket.org/free5gc-team/nef/pkg/factory"
	"bitbucket.org/free5gc-team/openapi"
	"bitbucket.org/free5gc-team/openapi/Nnrf_NFDiscovery"
	"bitbucket.org/free5gc-team/openapi/Nnrf_NFManagement"
	"bitbucket.org/free5gc-team/openapi/Npcf_PolicyAuthorization"
	"bitbucket.org/free5gc-team/openapi/Nudr_DataRepository"
	"bitbucket.org/free5gc-team/openapi/models"
)

type nef interface {
	Context() *nefctx.NefContext
	Config() *factory.Config
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
		nfDiscClients:   make(map[string]*Nnrf_NFDiscovery.APIClient),
		nfMngmntClients: make(map[string]*Nnrf_NFManagement.APIClient),
	}

	c.npcfService = &npcfService{
		consumer: c,
		clients:  make(map[string]*Npcf_PolicyAuthorization.APIClient),
	}

	c.nudrService = &nudrService{
		consumer: c,
		clients:  make(map[string]*Nudr_DataRepository.APIClient),
	}
	return c, nil
}

func handleAPIServiceResponseError(rsp *http.Response, err error) (int, interface{}) {
	var rspCode int
	var rspBody interface{}
	if rsp.Status != err.Error() {
		rspCode, rspBody = handleDeserializeError(rsp, err)
	} else {
		pd := err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails)
		rspCode, rspBody = int(pd.Status), &pd
	}
	return rspCode, rspBody
}

func handleDeserializeError(rsp *http.Response, err error) (int, interface{}) {
	logger.ConsumerLog.Errorf("Deserialize ProblemDetails Error: %s", err.Error())
	pd := &models.ProblemDetails{
		Status: int32(rsp.StatusCode),
		Detail: err.Error(),
	}
	return int(pd.Status), pd
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
