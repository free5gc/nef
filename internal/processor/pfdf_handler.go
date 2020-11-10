package processor

import (
	"net/http"

	"bitbucket.org/free5gc-team/nef/internal/logger"
	"bitbucket.org/free5gc-team/openapi/models"
)

func (p *Processor) GetApplicationsPFD() *HandlerResponse {
	logger.PFDFLog.Infof("GetApplicationsPFD")
	return &HandlerResponse{http.StatusOK, nil, nil}
}

func (p *Processor) GetIndividualApplicationPFD(appID string) *HandlerResponse {
	logger.PFDFLog.Infof("GetIndividualApplicationPFD - appID[%s]", appID)
	return &HandlerResponse{http.StatusOK, nil, nil}
}

func (p *Processor) PostPFDSubscriptions(pfdSubsc *models.PfdSubscription) *HandlerResponse {
	logger.PFDFLog.Infof("PostPFDSubscriptions")
	return &HandlerResponse{http.StatusOK, nil, nil}
}

func (p *Processor) DeleteIndividualPFDSubscription(subscID string) *HandlerResponse {
	logger.PFDFLog.Infof("DeleteIndividualPFDSubscription - subscID[%s]", subscID)
	return &HandlerResponse{http.StatusOK, nil, nil}
}
