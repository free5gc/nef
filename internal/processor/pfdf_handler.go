package processor

import (
	"net/http"

	"bitbucket.org/free5gc-team/nef/internal/logger"
	"bitbucket.org/free5gc-team/nef/internal/util"
	"bitbucket.org/free5gc-team/openapi/models"
)

func (p *Processor) GetApplicationsPFD(appIDs []string) *HandlerResponse {
	logger.PFDFLog.Infof("GetApplicationsPFD")

	// TODO: Support SupportedFeatures
	rspCode, rspBody := p.consumer.UdrSrv.AppDataPfdsGet(appIDs)

	// return &HandlerResponse{http.StatusOK, nil, pfdDataForApps}
	return &HandlerResponse{rspCode, nil, rspBody}
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
