package consumer

import (
	ctx "context"
	"net/http"
	"sync"

	"github.com/antihax/optional"

	"bitbucket.org/free5gc-team/nef/internal/context"
	"bitbucket.org/free5gc-team/nef/internal/factory"
	"bitbucket.org/free5gc-team/nef/internal/logger"
	"bitbucket.org/free5gc-team/openapi"
	"bitbucket.org/free5gc-team/openapi/Nnrf_NFDiscovery"
	"bitbucket.org/free5gc-team/openapi/Nudr_DataRepository"
	"bitbucket.org/free5gc-team/openapi/models"
)

type ConsumerUDRService struct {
	cfg            *factory.Config
	nefCtx         *context.NefContext
	nrfSrv         *ConsumerNRFService
	clientDataRepo *Nudr_DataRepository.APIClient
	clientMtx      sync.RWMutex
}

const ServiceName_NUDR_DR string = "nudr-dr"

func NewConsumerUDRService(nefCfg *factory.Config, nefCtx *context.NefContext,
	nrfSrv *ConsumerNRFService) *ConsumerUDRService {
	c := &ConsumerUDRService{cfg: nefCfg, nefCtx: nefCtx, nrfSrv: nrfSrv}

	return c
}

func (c *ConsumerUDRService) initDataRepoAPIClient() error {
	c.clientMtx.Lock()
	defer c.clientMtx.Unlock()

	if c.clientDataRepo != nil {
		return nil
	}

	param := Nnrf_NFDiscovery.SearchNFInstancesParamOpts{
		ServiceNames: optional.NewInterface([]string{ServiceName_NUDR_DR}),
	}
	uri, err := c.nrfSrv.SearchNFServiceUri("UDR", ServiceName_NUDR_DR, &param)
	if err != nil {
		return err
	}
	logger.ConsumerLog.Infof("initDataRepoAPIClient: uri[%s]", uri)

	//TODO: Subscribe NRF to notify service URI change

	drCfg := Nudr_DataRepository.NewConfiguration()
	drCfg.SetBasePath(uri)
	c.clientDataRepo = Nudr_DataRepository.NewAPIClient(drCfg)

	return nil
}

func (c *ConsumerUDRService) AppDataInfluenceDataPut(influenceID string,
	tiData *models.TrafficInfluData) (int, interface{}) {
	var (
		err     error
		rspCode int
		rspBody interface{}
		result  models.TrafficInfluData
		rsp     *http.Response
	)
	if err = c.initDataRepoAPIClient(); err != nil {
		goto END
	}

	c.clientMtx.RLock()
	result, rsp, err = c.clientDataRepo.DefaultApi.
		ApplicationDataInfluenceDataInfluenceIdPut(ctx.Background(), influenceID, *tiData)
	c.clientMtx.RUnlock()

	if rsp != nil {
		rspCode = rsp.StatusCode
		if rsp.StatusCode == http.StatusCreated { //TODO: check more status codes
			rspBody = &result
		} else if err != nil {
			if rsp.Status != err.Error() {
				logger.ConsumerLog.Errorf("Deserialize ProblemDetails Error: %s", err.Error())
				rspBody = &models.ProblemDetails{
					Status: int32(rsp.StatusCode),
					Detail: err.Error(),
				}
				goto END
			}
			pd := err.(openapi.GenericOpenAPIError).Model().(models.ProblemDetails)
			rspBody = &pd
		}
	} else {
		logger.ConsumerLog.Errorf("PostAppSessions: server no response")
		rspCode = http.StatusInternalServerError
		detail := "server no response"
		if err != nil {
			detail = err.Error()
		}
		rspBody = &models.ProblemDetails{
			Title:  "System failure",
			Status: http.StatusInternalServerError,
			Detail: detail,
			Cause:  "SYSTEM_FAILURE",
		}
	}

END:
	return rspCode, rspBody
}
