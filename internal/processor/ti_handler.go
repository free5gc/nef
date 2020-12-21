package processor

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"bitbucket.org/free5gc-team/nef/internal/context"
	"bitbucket.org/free5gc-team/nef/internal/factory"
	"bitbucket.org/free5gc-team/nef/internal/logger"
	"bitbucket.org/free5gc-team/nef/internal/util"
	"bitbucket.org/free5gc-team/openapi/models"
)

func (p *Processor) GetTrafficInfluenceSubscription(afID string) *HandlerResponse {
	logger.TrafInfluLog.Infof("GetTrafficInfluenceSubscription - afID[%s]", afID)

	var (
		tiSubList []models.TrafficInfluSub
		subInPCF  []string
		subInUDR  []string
	)

	afCtx := p.nefCtx.GetAfCtx(afID)
	if afCtx == nil {
		problemDetails := util.ProblemDetailsDataNotFound("Target AF is not existed")
		return &HandlerResponse{http.StatusNotFound, nil, problemDetails}
	}

	for _, subsc := range afCtx.GetAllSubsc() {
		if subsc.GetIsIndividualUEAddr() == true {
			subInPCF = append(subInPCF, subsc.GetAppSessID())
		} else {
			subInUDR = append(subInUDR, subsc.GetInfluenceID())
		}
	}

	if len(subInPCF) > 0 {
		for _, appSessID := range subInPCF {
			rspCode, rspBody := p.consumer.PcfSrv.GetAppSession(appSessID)
			if rspCode != http.StatusOK {
				return &HandlerResponse{rspCode, nil, rspBody}
			}
			tiSub := convertAppSessionContextToTrafficInfluSub(rspBody.(*models.AppSessionContext))
			// Not Complete: Need to advise the Self IE
			tiSub.Self = genTrafficInfluSubURI(p.cfg.GetSbiUri(), afID, appSessID)
			tiSubList = append(tiSubList, *tiSub)
		}
	}
	if len(subInUDR) > 0 {
		rspCode, rspBody := p.consumer.UdrSrv.AppDataInfluenceDataGet(subInUDR)
		if rspCode != http.StatusOK {
			return &HandlerResponse{rspCode, nil, rspBody}
		}

		for _, tiData := range *rspBody.(*[]models.TrafficInfluData) {
			tiSub := convertTrafficInfluDataToTrafficInfluSub(&tiData)
			// Not Complete: Need to advise the Self IE
			tiSub.Self = genTrafficInfluSubURI(p.cfg.GetSbiUri(), afID, "0")
			tiSubList = append(tiSubList, *tiSub)
		}
	}

	return &HandlerResponse{http.StatusOK, nil, &tiSubList}
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
		//Single UE, sent to PCF
		rsp = p.pcfPostAppSessions(afSubsc, tiSub)
	} else if len(tiSub.ExternalGroupId) > 0 || tiSub.AnyUeInd {
		//Group or any UE, sent to UDR
		influenceID := uuid.New().String()
		afSubsc.SetInfluenceID(influenceID)
		tiData := convertTrafficInfluSubToTrafficInfluData(tiSub)
		rspCode, rspBody := p.consumer.UdrSrv.AppDataInfluenceDataPut(influenceID, tiData)
		rsp = &HandlerResponse{
			Status: rspCode,
			Body:   rspBody,
		}
	} else {
		//Invalid case. Return Error
		pd := util.ProblemDetailsMalformedReqSyntax("Not individual UE case, nor group case")
		rsp = &HandlerResponse{
			Status: int(pd.Status),
			Body:   pd,
		}
	}

	if rsp.Status >= http.StatusOK && rsp.Status <= http.StatusAlreadyReported {
		p.nefCtx.AddAfCtx(afCtx)
		afCtx.AddSubsc(afSubsc)

		//Create Location URI
		locUri := p.cfg.GetSbiUri() + factory.TRAFF_INFLU_RES_URI_PREFIX + "/" + afID +
			"/subscriptions/" + afSubsc.GetSubscID()
		rsp.Headers = map[string][]string{
			"Location": {locUri},
		}
	}
	return rsp
}

func (p *Processor) GetIndividualTrafficInfluenceSubscription(afID, subscID string) *HandlerResponse {
	logger.TrafInfluLog.Infof("GetIndividualTrafficInfluenceSubscription - afID[%s], subscID[%s]", afID, subscID)

	afCtx := p.nefCtx.GetAfCtx(afID)
	if afCtx == nil {
		problemDetails := util.ProblemDetailsDataNotFound("Target AF is not existed")
		return &HandlerResponse{http.StatusNotFound, nil, problemDetails}
	}

	subsc := afCtx.GetSubsc(subscID)
	if afCtx == nil {
		problemDetails := util.ProblemDetailsDataNotFound("Target subscription is not existed")
		return &HandlerResponse{http.StatusNotFound, nil, problemDetails}
	}

	if subsc.GetIsIndividualUEAddr() {
		rspCode, rspBody := p.consumer.PcfSrv.GetAppSession(subsc.GetAppSessID())
		if rspCode != http.StatusOK {
			return &HandlerResponse{rspCode, nil, rspBody}
		}
		tiSub := convertAppSessionContextToTrafficInfluSub(rspBody.(*models.AppSessionContext))
		tiSub.Self = genTrafficInfluSubURI(p.cfg.GetSbiUri(), afID, subscID)
		return &HandlerResponse{http.StatusOK, nil, tiSub}
	} else {
		rspCode, rspBody := p.consumer.UdrSrv.AppDataInfluenceDataIdGet(subsc.GetInfluenceID())
		if rspCode != http.StatusOK {
			return &HandlerResponse{rspCode, nil, rspBody}
		}
		tiSub := convertTrafficInfluDataToTrafficInfluSub(rspBody.(*models.TrafficInfluData))
		tiSub.Self = genTrafficInfluSubURI(p.cfg.GetSbiUri(), afID, subscID)
		return &HandlerResponse{http.StatusOK, nil, tiSub}
	}

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
					//NotificationUri:
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

func genTrafficInfluSubURI(sbiURI, afID, subscriptionId string) string {
	// E.g. https://localhost:29505/3gpp-traffic-Influence/v1/{afId}/subscriptions/{subscriptionId}
	return fmt.Sprintf("%s%s/%s/subscriptions/%s",
		sbiURI, factory.TRAFF_INFLU_RES_URI_PREFIX, afID, subscriptionId)
}

func convertTrafficInfluDataToTrafficInfluSub(tiData *models.TrafficInfluData) *models.TrafficInfluSub {
	tiSub := &models.TrafficInfluSub{
		AppReloInd:        tiData.AppReloInd,
		AfAppId:           tiData.AfAppId,
		Dnn:               tiData.Dnn,
		EthTrafficFilters: tiData.EthTrafficFilters,
		Snssai:            tiData.Snssai,
		TrafficFilters:    tiData.TrafficFilters,
		TrafficRoutes:     tiData.TrafficRoutes,
	}

	return tiSub
}

func convertTrafficInfluSubToTrafficInfluData(tiSub *models.TrafficInfluSub) *models.TrafficInfluData {
	tiData := &models.TrafficInfluData{
		AppReloInd:        tiSub.AppReloInd,
		AfAppId:           tiSub.AfAppId,
		Dnn:               tiSub.Dnn,
		EthTrafficFilters: tiSub.EthTrafficFilters,
		Snssai:            tiSub.Snssai,
		TrafficFilters:    tiSub.TrafficFilters,
		TrafficRoutes:     tiSub.TrafficRoutes,
	}

	return tiData
}

func convertAppSessionContextToTrafficInfluSub(appSessionCtx *models.AppSessionContext) *models.TrafficInfluSub {
	tiSub := &models.TrafficInfluSub{
		AfAppId:     appSessionCtx.AscReqData.AfAppId,
		AppReloInd:  appSessionCtx.AscReqData.AfRoutReq.AppReloc,
		DnaiChgType: appSessionCtx.AscReqData.AfRoutReq.UpPathChgSub.DnaiChgType,
		Dnn:         appSessionCtx.AscReqData.Dnn,
		Gpsi:        appSessionCtx.AscReqData.Gpsi,
		SuppFeat:    appSessionCtx.AscReqData.SuppFeat,
	}

	return tiSub
}
