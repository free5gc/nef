package consumer

import (
	ctx "context"
	"net/http"
	"sync"

	"github.com/antihax/optional"

	"bitbucket.org/free5gc-team/nef/internal/context"
	"bitbucket.org/free5gc-team/nef/internal/factory"
	"bitbucket.org/free5gc-team/nef/internal/logger"
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

func (c *ConsumerUDRService) AppDataInfluenceDataGet(influenceIDs []string) (int, interface{}) {
	var (
		err     error
		rspCode int
		rspBody interface{}
		result  []models.TrafficInfluData
		rsp     *http.Response
	)

	if err = c.initDataRepoAPIClient(); err != nil {
		return rspCode, rspBody
	}

	param := &Nudr_DataRepository.ApplicationDataInfluenceDataGetParamOpts{
		InfluenceIds: optional.NewInterface(influenceIDs),
	}

	c.clientMtx.RLock()
	result, rsp, err = c.clientDataRepo.DefaultApi.
		ApplicationDataInfluenceDataGet(ctx.Background(), param)
	c.clientMtx.RUnlock()

	if rsp != nil {
		rspCode = rsp.StatusCode
		if rsp.StatusCode == http.StatusOK {
			rspBody = &result
		} else if err != nil {
			rspCode, rspBody = handleAPIServiceResponseError(rsp, err)
		}
	} else {
		//API Service Internal Error or Server No Response
		rspCode, rspBody = handleAPIServiceNoResponse(err)
	}

	return rspCode, rspBody
}

func (c *ConsumerUDRService) AppDataInfluenceDataIdGet(influenceID string) (int, interface{}) {
	var (
		err     error
		rspCode int
		rspBody interface{}
		result  []models.TrafficInfluData
		rsp     *http.Response
	)

	if err = c.initDataRepoAPIClient(); err != nil {
		return rspCode, rspBody
	}

	param := &Nudr_DataRepository.ApplicationDataInfluenceDataGetParamOpts{
		InfluenceIds: optional.NewInterface(influenceID),
	}

	c.clientMtx.RLock()
	result, rsp, err = c.clientDataRepo.DefaultApi.
		ApplicationDataInfluenceDataGet(ctx.Background(), param)
	c.clientMtx.RUnlock()

	if rsp != nil {
		rspCode = rsp.StatusCode
		if rsp.StatusCode == http.StatusOK {
			rspBody = &result
		} else if err != nil {
			rspCode, rspBody = handleAPIServiceResponseError(rsp, err)
		}
	} else {
		//API Service Internal Error or Server No Response
		rspCode, rspBody = handleAPIServiceNoResponse(err)
	}

	return rspCode, rspBody
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
		return rspCode, rspBody
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
			rspCode, rspBody = handleAPIServiceResponseError(rsp, err)
		}
	} else {
		//API Service Internal Error or Server No Response
		rspCode, rspBody = handleAPIServiceNoResponse(err)
	}

	return rspCode, rspBody
}

// TS 29.519 v15.3.0 6.2.3.3.1
func (c *ConsumerUDRService) AppDataPfdsGet(appID []string) (int, interface{}) {
	var (
		err     error
		rspCode int
		rspBody interface{}
		result  []models.PfdDataForApp
		rsp     *http.Response
	)

	if err = c.initDataRepoAPIClient(); err != nil {
		return rspCode, rspBody
	}

	param := &Nudr_DataRepository.ApplicationDataPfdsGetParamOpts{
		AppId: optional.NewInterface(appID),
	}

	c.clientMtx.RLock()
	result, rsp, err = c.clientDataRepo.DefaultApi.ApplicationDataPfdsGet(ctx.Background(), param)
	c.clientMtx.RUnlock()

	if rsp != nil {
		rspCode = rsp.StatusCode
		if rsp.StatusCode == http.StatusOK {
			rspBody = &result
		} else if err != nil {
			rspCode, rspBody = handleAPIServiceResponseError(rsp, err)
		}
	} else {
		//API Service Internal Error or Server No Response
		rspCode, rspBody = handleAPIServiceNoResponse(err)
	}

	return rspCode, rspBody
}

// TS 29.519 v15.3.0 6.2.4.3.3
func (c *ConsumerUDRService) AppDataPfdsAppIdPut(appID string, pfdDataForApp *models.PfdDataForApp) (int, interface{}) {
	var (
		err     error
		rspCode int
		rspBody interface{}
		result  models.PfdDataForApp
		rsp     *http.Response
	)

	if err = c.initDataRepoAPIClient(); err != nil {
		return rspCode, rspBody
	}

	c.clientMtx.RLock()
	result, rsp, err = c.clientDataRepo.DefaultApi.ApplicationDataPfdsAppIdPut(ctx.Background(), appID, *pfdDataForApp)
	c.clientMtx.RUnlock()

	if rsp != nil {
		rspCode = rsp.StatusCode
		if rsp.StatusCode == http.StatusOK || rsp.StatusCode == http.StatusCreated {
			rspBody = &result
		} else if err != nil {
			rspCode, rspBody = handleAPIServiceResponseError(rsp, err)
		}
	} else {
		//API Service Internal Error or Server No Response
		rspCode, rspBody = handleAPIServiceNoResponse(err)
	}

	return rspCode, rspBody
}

// TS 29.519 v15.3.0 6.2.4.3.2
func (c *ConsumerUDRService) AppDataPfdsAppIdDelete(appID string) (int, interface{}) {
	var (
		err     error
		rspCode int
		rspBody interface{}
		rsp     *http.Response
	)

	if err = c.initDataRepoAPIClient(); err != nil {
		return rspCode, rspBody
	}

	c.clientMtx.RLock()
	rsp, err = c.clientDataRepo.DefaultApi.ApplicationDataPfdsAppIdDelete(ctx.Background(), appID)
	c.clientMtx.RUnlock()

	if rsp != nil {
		rspCode = rsp.StatusCode
		if err != nil {
			rspCode, rspBody = handleAPIServiceResponseError(rsp, err)
		}
	} else {
		//API Service Internal Error or Server No Response
		rspCode, rspBody = handleAPIServiceNoResponse(err)
	}

	return rspCode, rspBody
}

// TS 29.519 v15.3.0 6.2.4.3.1
func (c *ConsumerUDRService) AppDataPfdsAppIdGet(appID string) (int, interface{}) {
	var (
		err     error
		rspCode int
		rspBody interface{}
		result  models.PfdDataForApp
		rsp     *http.Response
	)

	if err = c.initDataRepoAPIClient(); err != nil {
		return rspCode, rspBody
	}

	c.clientMtx.RLock()
	result, rsp, err = c.clientDataRepo.DefaultApi.ApplicationDataPfdsAppIdGet(ctx.Background(), appID)
	c.clientMtx.RUnlock()

	if rsp != nil {
		rspCode = rsp.StatusCode
		if rsp.StatusCode == http.StatusOK {
			rspBody = &result
		} else if err != nil {
			rspCode, rspBody = handleAPIServiceResponseError(rsp, err)
		}
	} else {
		//API Service Internal Error or Server No Response
		rspCode, rspBody = handleAPIServiceNoResponse(err)
	}

	return rspCode, rspBody
}

func (c *ConsumerUDRService) AppDataInfluenceDataPatch(influenceID string, tiSubPatch *models.TrafficInfluDataPatch) (int, interface{}) {
	var (
		err     error
		rspCode int
		rspBody interface{}
		result  models.TrafficInfluData
		rsp     *http.Response
	)

	if err = c.initDataRepoAPIClient(); err != nil {
		return rspCode, rspBody
	}

	c.clientMtx.RLock()
	result, rsp, err = c.clientDataRepo.DefaultApi.
		ApplicationDataInfluenceDataInfluenceIdPatch(ctx.Background(), influenceID, *tiSubPatch)
	c.clientMtx.RUnlock()

	if rsp != nil {
		rspCode = rsp.StatusCode
		if rsp.StatusCode == http.StatusOK {
			rspBody = &result
		} else if err != nil {
			rspCode, rspBody = handleAPIServiceResponseError(rsp, err)
		}
	} else {
		//API Service Internal Error or Server No Response
		rspCode, rspBody = handleAPIServiceNoResponse(err)
	}

	return rspCode, rspBody
}

func (c *ConsumerUDRService) AppDataInfluenceDataDelete(influenceID string) (int, interface{}) {
	var (
		err     error
		rspCode int
		rspBody interface{}
		rsp     *http.Response
	)

	if err = c.initDataRepoAPIClient(); err != nil {
		return rspCode, rspBody
	}

	c.clientMtx.RLock()
	rsp, err = c.clientDataRepo.DefaultApi.
		ApplicationDataInfluenceDataInfluenceIdDelete(ctx.Background(), influenceID)
	c.clientMtx.RUnlock()

	if rsp != nil {
		rspCode = rsp.StatusCode
		if rsp.StatusCode == http.StatusOK {
			rspBody = &rsp.Body
		} else if err != nil {
			rspCode, rspBody = handleAPIServiceResponseError(rsp, err)
		}
	} else {
		//API Service Internal Error or Server No Response
		rspCode, rspBody = handleAPIServiceNoResponse(err)
	}

	return rspCode, rspBody
}