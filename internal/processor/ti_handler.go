package processor

import (
	"net/http"

	"github.com/google/uuid"

	"bitbucket.org/free5gc-team/nef/internal/context"
	"bitbucket.org/free5gc-team/nef/internal/logger"
	"bitbucket.org/free5gc-team/nef/pkg/factory"
	"bitbucket.org/free5gc-team/openapi"
	"bitbucket.org/free5gc-team/openapi/models"
)

func (p *Processor) GetTrafficInfluenceSubscription(afID string) *HandlerResponse {
	logger.TrafInfluLog.Infof("GetTrafficInfluenceSubscription - afID[%s]", afID)
	return &HandlerResponse{http.StatusOK, nil, nil}
}

func (p *Processor) PostTrafficInfluenceSubscription(afID string,
	tiSub *models.TrafficInfluSub) *HandlerResponse {
	var rsp *HandlerResponse
	logger.TrafInfluLog.Infof("PostTrafficInfluenceSubscription - afID[%s]", afID)

	rsp = validateTrafficInfluenceData(tiSub)
	if rsp != nil {
		return rsp
	}

	afCtx := p.nefCtx.NewAfCtx(afID)
	afSubsc := p.nefCtx.NewAfSubsc(afCtx)
	if len(tiSub.Gpsi) > 0 || len(tiSub.Ipv4Addr) > 0 || len(tiSub.Ipv6Addr) > 0 {
		// Single UE, sent to PCF
		rsp = p.pcfPostAppSessions(afSubsc, tiSub)
	} else if len(tiSub.ExternalGroupId) > 0 || tiSub.AnyUeInd {
		// Group or any UE, sent to UDR
		afSubsc.SetInfluenceID(uuid.New().String())
		rsp = p.udrPutAppData(afSubsc, tiSub)
	} else {
		// Invalid case. Return Error
		pd := openapi.ProblemDetailsMalformedReqSyntax("Not individual UE case, nor group case")
		rsp = &HandlerResponse{
			Status: int(pd.Status),
			Body:   pd,
		}
	}

	if rsp.Status >= http.StatusOK && rsp.Status <= http.StatusAlreadyReported {
		p.nefCtx.AddAfCtx(afCtx)
		afCtx.AddSubsc(afSubsc)

		// Create Location URI
		locUri := p.cfg.SbiUri() + factory.TraffInfluResUriPrefix + "/" + afID +
			"/subscriptions/" + afSubsc.GetSubscID()
		rsp.Headers = map[string][]string{
			"Location": {locUri},
		}
	}
	return rsp
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

func validateTrafficInfluenceData(tiSub *models.TrafficInfluSub) *HandlerResponse {
	return nil
}

func (p *Processor) pcfPostAppSessions(afSubsc *context.AfSubscription,
	tiSub *models.TrafficInfluSub) *HandlerResponse {
	asc := models.AppSessionContext{
		AscReqData: &models.AppSessionContextReqData{
			AfAppId: tiSub.AfAppId,
			AfRoutReq: &models.AfRoutingRequirement{
				AppReloc: tiSub.AppReloInd,
				UpPathChgSub: &models.UpPathChgEvent{
					DnaiChgType: tiSub.DnaiChgType,
					// NotificationUri:
				},
			},
		},
	}

	rspCode, rspBody, appSessID := p.consumer.PcfSrv.PostAppSessions(&asc)
	if rspCode == http.StatusCreated {
		afSubsc.SetAppSessID(appSessID)
		return &HandlerResponse{rspCode, nil, nil}
	}
	return &HandlerResponse{rspCode, nil, rspBody}
}

func (p *Processor) udrPutAppData(afSubsc *context.AfSubscription, tiSub *models.TrafficInfluSub) *HandlerResponse {
	return nil
}
