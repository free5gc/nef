package consumer

import (
	"net/http"

	"bitbucket.org/free5gc-team/nef/internal/context"
	"bitbucket.org/free5gc-team/nef/internal/logger"
	"bitbucket.org/free5gc-team/openapi"
	"bitbucket.org/free5gc-team/openapi/models"
)

type Consumer struct {
	NrfSrv *ConsumerNRFService
	PcfSrv *ConsumerPCFService
	UdrSrv *ConsumerUDRService
}

func NewConsumer(nefCtx *context.NefContext) (*Consumer, error) {
	var err error
	c := &Consumer{}
	if c.NrfSrv, err = NewConsumerNRFService(nefCtx); err != nil {
		return nil, err
	}
	if c.PcfSrv, err = NewConsumerPCFService(nefCtx, c.NrfSrv); err != nil {
		return nil, err
	}
	if c.UdrSrv, err = NewConsumerUDRService(nefCtx, c.NrfSrv); err != nil {
		return nil, err
	}
	c.NrfSrv.RegisterNFInstance()

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
