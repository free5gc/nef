package consumer

import (
	ctx "context"
	"net/http"
	"strings"
	"sync"

	"github.com/antihax/optional"

	"bitbucket.org/free5gc-team/nef/internal/context"
	"bitbucket.org/free5gc-team/nef/internal/factory"
	"bitbucket.org/free5gc-team/nef/internal/logger"
	"bitbucket.org/free5gc-team/openapi"
	"bitbucket.org/free5gc-team/openapi/Nnrf_NFDiscovery"
	"bitbucket.org/free5gc-team/openapi/Npcf_PolicyAuthorization"
	"bitbucket.org/free5gc-team/openapi/models"
)

type ConsumerPCFService struct {
	cfg              *factory.Config
	nefCtx           *context.NefContext
	nrfSrv           *ConsumerNRFService
	clientPolicyAuth *Npcf_PolicyAuthorization.APIClient
	clientMtx        sync.RWMutex
}

const ServiceName_NPCF_POLICYAUTHORIZATION string = "npcf-policyauthorization"

func NewConsumerPCFService(nefCfg *factory.Config, nefCtx *context.NefContext,
	nrfSrv *ConsumerNRFService) *ConsumerPCFService {
	c := &ConsumerPCFService{cfg: nefCfg, nefCtx: nefCtx, nrfSrv: nrfSrv}

	return c
}

func (c *ConsumerPCFService) initPolicyAuthAPIClient() error {
	c.clientMtx.Lock()
	defer c.clientMtx.Unlock()

	if c.clientPolicyAuth != nil {
		return nil
	}

	param := Nnrf_NFDiscovery.SearchNFInstancesParamOpts{
		ServiceNames: optional.NewInterface([]string{ServiceName_NPCF_POLICYAUTHORIZATION}),
	}
	uri, err := c.nrfSrv.SearchNFServiceUri("PCF", ServiceName_NPCF_POLICYAUTHORIZATION, &param)
	if err != nil {
		logger.ConsumerLog.Errorf(err.Error())
		return err
	}
	logger.ConsumerLog.Infof("initPolicyAuthAPIClient: uri[%s]", uri)

	paCfg := Npcf_PolicyAuthorization.NewConfiguration()
	paCfg.SetBasePath(uri)
	c.clientPolicyAuth = Npcf_PolicyAuthorization.NewAPIClient(paCfg)
	return nil
}

func (c *ConsumerPCFService) PostAppSessions(asc *models.AppSessionContext) (int, interface{}, string) {
	var (
		err       error
		rspCode   int
		rspBody   interface{}
		appSessID string
		result    models.AppSessionContext
		rsp       *http.Response
	)

	if err = c.initPolicyAuthAPIClient(); err != nil {
		goto END
	}

	c.clientMtx.RLock()
	result, rsp, err = c.clientPolicyAuth.ApplicationSessionsCollectionApi.PostAppSessions(ctx.Background(), *asc)
	c.clientMtx.RUnlock()

	if rsp != nil {
		rspCode = rsp.StatusCode
		if rsp.StatusCode == http.StatusCreated {
			logger.ConsumerLog.Debugf("PostAppSessions RspData: %+v", result)
			rspBody = &result
			appSessID = rsp.Header.Get("Location")
			if strings.Contains(appSessID, "http") {
				index := strings.LastIndex(appSessID, "/")
				appSessID = appSessID[index+1:]
			}
			logger.ConsumerLog.Infof("PostAppSessions appSessID[%s]", appSessID)
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
	return rspCode, rspBody, appSessID
}
