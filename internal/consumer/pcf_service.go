package consumer

import (
	ctx "context"
	"net/http"
	"strings"
	"sync"

	"github.com/antihax/optional"

	"bitbucket.org/free5gc-team/nef/internal/logger"
	"bitbucket.org/free5gc-team/openapi/Npcf_PolicyAuthorization"
	"bitbucket.org/free5gc-team/openapi/models"
)

type npcfService struct {
	consumer *Consumer

	mu      sync.RWMutex
	clients map[string]*Npcf_PolicyAuthorization.APIClient
}

func (s *npcfService) getClient(uri string) *Npcf_PolicyAuthorization.APIClient {
	s.mu.RLock()
	if client, ok := s.clients[uri]; ok {
		defer s.mu.RUnlock()
		return client
	} else {
		configuration := Npcf_PolicyAuthorization.NewConfiguration()
		configuration.SetBasePath(uri)
		cli := Npcf_PolicyAuthorization.NewAPIClient(configuration)

		s.mu.RUnlock()
		s.mu.Lock()
		defer s.mu.Unlock()
		s.clients[uri] = cli
		return cli
	}
}

func (s *npcfService) getPcfPolicyAuthUri() (string, error) {
	uri := s.consumer.Context().PcfPaUri()
	if uri == "" {
		sUri, err := s.consumer.nnrfService.SearchPcfPolicyAuthUri()
		if err == nil {
			s.consumer.Context().SetPcfPaUri(sUri)
		}
		return sUri, err
	}
	return uri, nil
}

func (s *npcfService) GetAppSession(appSessionId string) (int, interface{}) {
	var (
		err     error
		rspCode int
		rspBody interface{}
		result  models.AppSessionContext
		rsp     *http.Response
	)

	uri, err := s.getPcfPolicyAuthUri()
	if err != nil {
		return rspCode, rspBody
	}
	client := s.getClient(uri)

	result, rsp, err = client.IndividualApplicationSessionContextDocumentApi.
		GetAppSession(ctx.Background(), appSessionId)

	if rsp != nil {
		rspCode = rsp.StatusCode
		if rsp.StatusCode == http.StatusOK {
			rspBody = &result
		} else if err != nil {
			rspCode, rspBody = handleAPIServiceResponseError(rsp, err)
		}
	} else {
		// API Service Internal Error or Server No Response
		rspCode, rspBody = handleAPIServiceNoResponse(err)
	}

	return rspCode, rspBody
}

func (s *npcfService) PostAppSessions(asc *models.AppSessionContext) (int, interface{}, string) {
	var (
		err       error
		rspCode   int
		rspBody   interface{}
		appSessID string
		result    models.AppSessionContext
		rsp       *http.Response
	)

	uri, err := s.getPcfPolicyAuthUri()
	if err != nil {
		return rspCode, rspBody, appSessID
	}
	client := s.getClient(uri)

	result, rsp, err = client.ApplicationSessionsCollectionApi.PostAppSessions(ctx.TODO(), *asc)
	if rsp != nil {
		rspCode = rsp.StatusCode
		if rsp.StatusCode == http.StatusCreated {
			logger.ConsumerLog.Debugf("PostAppSessions RspData: %+v", result)
			rspBody = &result
			appSessID = getAppSessIDFromRspLocationHeader(rsp)
		} else if err != nil {
			rspCode, rspBody = handleAPIServiceResponseError(rsp, err)
		}
	} else {
		// API Service Internal Error or Server No Response
		rspCode, rspBody = handleAPIServiceNoResponse(err)
	}

	return rspCode, rspBody, appSessID
}

func (s *npcfService) PutAppSession(appSessionId string,
	ascUpdateData *models.AppSessionContextUpdateData,
	asc *models.AppSessionContext) (int, interface{}, string) {
	var (
		err       error
		rspCode   int
		rspBody   interface{}
		appSessID string
		result    models.AppSessionContext
		rsp       *http.Response
	)

	uri, err := s.getPcfPolicyAuthUri()
	if err != nil {
		return rspCode, rspBody, appSessID
	}
	client := s.getClient(uri)

	appSessID = appSessionId
	result, rsp, err = client.IndividualApplicationSessionContextDocumentApi.
		GetAppSession(ctx.Background(), appSessionId)

	if rsp != nil {
		if rsp.Body != nil {
			if bodyCloseErr := rsp.Body.Close(); bodyCloseErr != nil {
				logger.ConsumerLog.Errorf("SearchNFInstances err: response body cannot close: %+v", bodyCloseErr)
			}
		}

		rspCode = rsp.StatusCode
		if rsp.StatusCode == http.StatusOK {
			// Patch
			result, rsp, err = client.IndividualApplicationSessionContextDocumentApi.ModAppSession(
				ctx.Background(), appSessionId, *ascUpdateData)

			if rsp != nil {
				rspCode = rsp.StatusCode
				if rsp.StatusCode == http.StatusOK {
					logger.ConsumerLog.Debugf("PatchAppSessions RspData: %+v", result)
					rspBody = &result
				} else if err != nil {
					rspCode, rspBody = handleAPIServiceResponseError(rsp, err)
				}
			} else {
				// API Service Internal Error or Server No Response
				rspCode, rspBody = handleAPIServiceNoResponse(err)
			}

			return rspCode, rspBody, appSessID
		}
		// TODO:
		// else if err != nil {
		// 	// Post
		// }
	} else {
		// API Service Internal Error or Server No Response
		rspCode, rspBody = handleAPIServiceNoResponse(err)
		return rspCode, rspBody, appSessID
	}

	return rspCode, rspBody, appSessID
}

func (s *npcfService) PatchAppSession(appSessionId string,
	ascUpdateData *models.AppSessionContextUpdateData) (int, interface{}) {
	var (
		err     error
		rspCode int
		rspBody interface{}
		result  models.AppSessionContext
		rsp     *http.Response
	)

	uri, err := s.getPcfPolicyAuthUri()
	if err != nil {
		return rspCode, rspBody
	}
	client := s.getClient(uri)

	result, rsp, err = client.IndividualApplicationSessionContextDocumentApi.ModAppSession(
		ctx.Background(), appSessionId, *ascUpdateData)

	if rsp != nil {
		rspCode = rsp.StatusCode
		if rsp.StatusCode == http.StatusOK {
			logger.ConsumerLog.Debugf("PatchAppSessions RspData: %+v", result)
			rspBody = &result
		} else if err != nil {
			rspCode, rspBody = handleAPIServiceResponseError(rsp, err)
		}
	} else {
		// API Service Internal Error or Server No Response
		rspCode, rspBody = handleAPIServiceNoResponse(err)
	}

	return rspCode, rspBody
}

func (s *npcfService) DeleteAppSession(appSessionId string) (int, interface{}) {
	var (
		err     error
		rspCode int
		rspBody interface{}
		result  models.AppSessionContext
		rsp     *http.Response
	)

	uri, err := s.getPcfPolicyAuthUri()
	if err != nil {
		return rspCode, rspBody
	}
	client := s.getClient(uri)

	param := &Npcf_PolicyAuthorization.DeleteAppSessionParamOpts{
		EventsSubscReqData: optional.NewInterface(models.EventsSubscReqData{}),
	}

	result, rsp, err = client.IndividualApplicationSessionContextDocumentApi.DeleteAppSession(
		ctx.Background(), appSessionId, param)

	if rsp != nil {
		rspCode = rsp.StatusCode
		if rsp.StatusCode == http.StatusOK {
			logger.ConsumerLog.Debugf("DeleteAppSessions RspData: %+v", result)
			rspBody = &result
		} else if err != nil {
			rspCode, rspBody = handleAPIServiceResponseError(rsp, err)
		}
	} else {
		// API Service Internal Error or Server No Response
		rspCode, rspBody = handleAPIServiceNoResponse(err)
	}

	return rspCode, rspBody
}

func getAppSessIDFromRspLocationHeader(rsp *http.Response) string {
	appSessID := ""
	loc := rsp.Header.Get("Location")
	if strings.Contains(loc, "http") {
		index := strings.LastIndex(loc, "/")
		appSessID = loc[index+1:]
	}
	logger.ConsumerLog.Infof("appSessID=%q", appSessID)
	return appSessID
}
