package processor

import (
	"net/http"

	"bitbucket.org/free5gc-team/nef/internal/logger"
	"bitbucket.org/free5gc-team/openapi/models"
)

func (p *Processor) GetTrafficInfluenceSubscription(afID string) *HandlerResponse {
	logger.TrafInfluLog.Infof("GetTrafficInfluenceSubscription - afID[%s]", afID)
	return &HandlerResponse{http.StatusOK, nil, nil}
}

func (p *Processor) PostTrafficInfluenceSubscription(afID string,
	tiSub *models.TrafficInfluSub) *HandlerResponse {
	logger.TrafInfluLog.Infof("PostTrafficInfluenceSubscription - afID[%s]", afID)
	return &HandlerResponse{http.StatusOK, nil, nil}
}

func (p *Processor) GetIndividualTrafficInfluenceSubscription(afID, subscID string) *HandlerResponse {
	logger.TrafInfluLog.Infof("GetIndividualTrafficInfluenceSubscription - afID[%s], subscID[%s]", afID, subscID)
	return &HandlerResponse{http.StatusOK, nil, nil}
}

func (p *Processor) PutIndividualTrafficInfluenceSubscription(afID, subscID string,
	tiSub *models.TrafficInfluSub) *HandlerResponse {
	logger.TrafInfluLog.Infof("PutIndividualTrafficInfluenceSubscription - afID[%s], subscID[%s]", afID, subscID)
	return &HandlerResponse{http.StatusOK, nil, nil}
}

func (p *Processor) PatchIndividualTrafficInfluenceSubscription(afID, subscID string,
	tiSubPatch *models.TrafficInfluSubPatch) *HandlerResponse {
	logger.TrafInfluLog.Infof("PatchIndividualTrafficInfluenceSubscription - afID[%s], subscID[%s]", afID, subscID)
	return &HandlerResponse{http.StatusOK, nil, nil}
}

func (p *Processor) DeleteIndividualTrafficInfluenceSubscription(afID, subscID string) *HandlerResponse {
	logger.TrafInfluLog.Infof("DeleteIndividualTrafficInfluenceSubscription - afID[%s], subscID[%s]", afID, subscID)
	return &HandlerResponse{http.StatusOK, nil, nil}
}
