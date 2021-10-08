package processor

import (
	"fmt"
	"net/http"

	"bitbucket.org/free5gc-team/nef/internal/logger"
	"bitbucket.org/free5gc-team/nef/pkg/factory"
	"bitbucket.org/free5gc-team/openapi"
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

	// TODO: Support SupportedFeatures
	rspCode, rspBody := p.consumer.UdrSrv.AppDataPfdsAppIdGet(appID)

	return &HandlerResponse{rspCode, nil, rspBody}
}

func (p *Processor) PostPFDSubscriptions(pfdSubsc *models.PfdSubscription) *HandlerResponse {
	logger.PFDFLog.Infof("PostPFDSubscriptions")

	// TODO: Support SupportedFeatures
	if len(pfdSubsc.NotifyUri) == 0 {
		return &HandlerResponse{http.StatusNotFound, nil, openapi.ProblemDetailsDataNotFound("Absent of Notify URI")}
	}

	subID := p.notifier.PfdChangeNotifier.AddPfdSub(pfdSubsc)
	hdrs := make(map[string][]string)
	addLocationheader(hdrs, genPfdSubscriptionURI(p.cfg.SbiUri(), subID))

	return &HandlerResponse{http.StatusCreated, hdrs, pfdSubsc}
}

func (p *Processor) DeleteIndividualPFDSubscription(subscID string) *HandlerResponse {
	logger.PFDFLog.Infof("DeleteIndividualPFDSubscription - subscID[%s]", subscID)

	if err := p.notifier.PfdChangeNotifier.DeletePfdSub(subscID); err != nil {
		return &HandlerResponse{http.StatusNotFound, nil, openapi.ProblemDetailsDataNotFound(err.Error())}
	}

	return &HandlerResponse{http.StatusNoContent, nil, nil}
}

func genPfdSubscriptionURI(sbiURI, subID string) string {
	// E.g. "https://localhost:29505/nnef-pfdmanagement/v1/subscriptions/{subscriptionId}
	return fmt.Sprintf("%s%s/subscriptions/%s", sbiURI, factory.NefPfdMngResUriPrefix, subID)
}
