package consumer

import (
	"net/http"

	"bitbucket.org/free5gc-team/nef/internal/context"
	"bitbucket.org/free5gc-team/nef/internal/factory"
	"bitbucket.org/free5gc-team/nef/internal/logger"
	"bitbucket.org/free5gc-team/nef/internal/util"
	"bitbucket.org/free5gc-team/openapi"
	"bitbucket.org/free5gc-team/openapi/models"
)

type Consumer struct {
	NrfSrv *ConsumerNRFService
	PcfSrv *ConsumerPCFService
	UdrSrv *ConsumerUDRService
}

func NewConsumer(nefCfg *factory.Config, nefCtx *context.NefContext) *Consumer {
	c := &Consumer{}
	if c.NrfSrv = NewConsumerNRFService(nefCfg, nefCtx); c.NrfSrv == nil {
		return nil
	}
	if c.PcfSrv = NewConsumerPCFService(nefCfg, nefCtx, c.NrfSrv); c.PcfSrv == nil {
		return nil
	}
	if c.UdrSrv = NewConsumerUDRService(nefCfg, nefCtx, c.NrfSrv); c.UdrSrv == nil {
		return nil
	}
	c.NrfSrv.RegisterNFInstance()

	return c
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
	pd := util.ProblemDetailsSystemFailure(detail)
	return int(pd.Status), pd
}
