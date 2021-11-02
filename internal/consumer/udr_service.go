package consumer

import (
	ctx "context"
	"net/http"
	"sync"

	"github.com/antihax/optional"

	"bitbucket.org/free5gc-team/openapi/Nudr_DataRepository"
	"bitbucket.org/free5gc-team/openapi/models"
)

type nudrService struct {
	consumer *Consumer

	mu      sync.RWMutex
	clients map[string]*Nudr_DataRepository.APIClient
}

func (s *nudrService) getClient(uri string) *Nudr_DataRepository.APIClient {
	s.mu.RLock()
	if client, ok := s.clients[uri]; ok {
		defer s.mu.RUnlock()
		return client
	} else {
		configuration := Nudr_DataRepository.NewConfiguration()
		configuration.SetBasePath(uri)
		cli := Nudr_DataRepository.NewAPIClient(configuration)

		s.mu.RUnlock()
		s.mu.Lock()
		defer s.mu.Unlock()
		s.clients[uri] = cli
		return cli
	}
}

func (s *nudrService) getUdrDrUri() (string, error) {
	uri := s.consumer.Context().UdrDrUri()
	if uri == "" {
		sUri, err := s.consumer.nnrfService.SearchUdrDrUri()
		if err == nil {
			s.consumer.Context().SetUdrDrUri(sUri)
		}
		return sUri, err
	}
	return uri, nil
}

func (s *nudrService) AppDataInfluenceDataGet(influenceIDs []string) (int, interface{}) {
	var (
		err     error
		rspCode int
		rspBody interface{}
		result  []models.TrafficInfluData
		rsp     *http.Response
	)

	uri, err := s.getUdrDrUri()
	if err != nil {
		return rspCode, rspBody
	}
	client := s.getClient(uri)

	param := &Nudr_DataRepository.ApplicationDataInfluenceDataGetParamOpts{
		InfluenceIds: optional.NewInterface(influenceIDs),
	}

	result, rsp, err = client.DefaultApi.
		ApplicationDataInfluenceDataGet(ctx.Background(), param)

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

func (s *nudrService) AppDataInfluenceDataIdGet(influenceID string) (int, interface{}) {
	var (
		err     error
		rspCode int
		rspBody interface{}
		result  []models.TrafficInfluData
		rsp     *http.Response
	)

	uri, err := s.getUdrDrUri()
	if err != nil {
		return rspCode, rspBody
	}
	client := s.getClient(uri)

	param := &Nudr_DataRepository.ApplicationDataInfluenceDataGetParamOpts{
		InfluenceIds: optional.NewInterface(influenceID),
	}

	result, rsp, err = client.DefaultApi.
		ApplicationDataInfluenceDataGet(ctx.Background(), param)

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

func (s *nudrService) AppDataInfluenceDataPut(influenceID string,
	tiData *models.TrafficInfluData) (int, interface{}) {
	var (
		err     error
		rspCode int
		rspBody interface{}
		result  models.TrafficInfluData
		rsp     *http.Response
	)

	uri, err := s.getUdrDrUri()
	if err != nil {
		return rspCode, rspBody
	}
	client := s.getClient(uri)

	result, rsp, err = client.DefaultApi.
		ApplicationDataInfluenceDataInfluenceIdPut(ctx.TODO(), influenceID, *tiData)

	if rsp != nil {
		rspCode = rsp.StatusCode
		if rsp.StatusCode == http.StatusCreated { // TODO: check more status codes
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

// TS 29.519 v15.3.0 6.2.3.3.1
func (s *nudrService) AppDataPfdsGet(appID []string) (int, interface{}) {
	var (
		err     error
		rspCode int
		rspBody interface{}
		result  []models.PfdDataForApp
		rsp     *http.Response
	)

	uri, err := s.getUdrDrUri()
	if err != nil {
		return rspCode, rspBody
	}
	client := s.getClient(uri)

	param := &Nudr_DataRepository.ApplicationDataPfdsGetParamOpts{
		AppId: optional.NewInterface(appID),
	}

	result, rsp, err = client.DefaultApi.ApplicationDataPfdsGet(ctx.TODO(), param)

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

// TS 29.519 v15.3.0 6.2.4.3.3
func (s *nudrService) AppDataPfdsAppIdPut(appID string, pfdDataForApp *models.PfdDataForApp) (int, interface{}) {
	var (
		err     error
		rspCode int
		rspBody interface{}
		result  models.PfdDataForApp
		rsp     *http.Response
	)

	uri, err := s.getUdrDrUri()
	if err != nil {
		return rspCode, rspBody
	}
	client := s.getClient(uri)

	result, rsp, err = client.DefaultApi.ApplicationDataPfdsAppIdPut(ctx.TODO(), appID, *pfdDataForApp)

	if rsp != nil {
		rspCode = rsp.StatusCode
		if rsp.StatusCode == http.StatusOK || rsp.StatusCode == http.StatusCreated {
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

// TS 29.519 v15.3.0 6.2.4.3.2
func (s *nudrService) AppDataPfdsAppIdDelete(appID string) (int, interface{}) {
	var (
		err     error
		rspCode int
		rspBody interface{}
		rsp     *http.Response
	)

	uri, err := s.getUdrDrUri()
	if err != nil {
		return rspCode, rspBody
	}
	client := s.getClient(uri)

	rsp, err = client.DefaultApi.ApplicationDataPfdsAppIdDelete(ctx.TODO(), appID)

	if rsp != nil {
		rspCode = rsp.StatusCode
		if err != nil {
			rspCode, rspBody = handleAPIServiceResponseError(rsp, err)
		}
	} else {
		// API Service Internal Error or Server No Response
		rspCode, rspBody = handleAPIServiceNoResponse(err)
	}

	return rspCode, rspBody
}

// TS 29.519 v15.3.0 6.2.4.3.1
func (s *nudrService) AppDataPfdsAppIdGet(appID string) (int, interface{}) {
	var (
		err     error
		rspCode int
		rspBody interface{}
		result  models.PfdDataForApp
		rsp     *http.Response
	)

	uri, err := s.getUdrDrUri()
	if err != nil {
		return rspCode, rspBody
	}
	client := s.getClient(uri)

	result, rsp, err = client.DefaultApi.ApplicationDataPfdsAppIdGet(ctx.TODO(), appID)

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

func (s *nudrService) AppDataInfluenceDataPatch(
	influenceID string, tiSubPatch *models.TrafficInfluDataPatch) (int, interface{}) {
	var (
		err     error
		rspCode int
		rspBody interface{}
		result  models.TrafficInfluData
		rsp     *http.Response
	)

	uri, err := s.getUdrDrUri()
	if err != nil {
		return rspCode, rspBody
	}
	client := s.getClient(uri)

	result, rsp, err = client.DefaultApi.
		ApplicationDataInfluenceDataInfluenceIdPatch(ctx.Background(), influenceID, *tiSubPatch)

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

func (s *nudrService) AppDataInfluenceDataDelete(influenceID string) (int, interface{}) {
	var (
		err     error
		rspCode int
		rspBody interface{}
		rsp     *http.Response
	)

	uri, err := s.getUdrDrUri()
	if err != nil {
		return rspCode, rspBody
	}
	client := s.getClient(uri)

	rsp, err = client.DefaultApi.
		ApplicationDataInfluenceDataInfluenceIdDelete(ctx.Background(), influenceID)

	if rsp != nil {
		rspCode = rsp.StatusCode
		if rsp.StatusCode == http.StatusOK {
			rspBody = &rsp.Body
		} else if err != nil {
			rspCode, rspBody = handleAPIServiceResponseError(rsp, err)
		}
	} else {
		// API Service Internal Error or Server No Response
		rspCode, rspBody = handleAPIServiceNoResponse(err)
	}

	return rspCode, rspBody
}
