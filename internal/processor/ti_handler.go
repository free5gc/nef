package processor

import (
	"fmt"
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

	var (
		tiSubList []models.TrafficInfluSub
		subInPCF  []string
		subInUDR  []string
	)

	afCtx := p.Context().GetAfCtx(afID)
	if afCtx == nil {
		problemDetails := openapi.ProblemDetailsDataNotFound("Target AF is not existed")
		return &HandlerResponse{http.StatusNotFound, nil, problemDetails}
	}

	for _, subsc := range afCtx.GetAllSubsc() {
		if subsc.IsIndividualUEAddr() {
			subInPCF = append(subInPCF, subsc.GetAppSessID())
		} else {
			subInUDR = append(subInUDR, subsc.GetInfluenceID())
		}
	}

	if len(subInPCF) > 0 {
		for _, appSessID := range subInPCF {
			rspCode, rspBody := p.Consumer().GetAppSession(appSessID)
			if rspCode != http.StatusOK {
				return &HandlerResponse{rspCode, nil, rspBody}
			}
			tiSub := convertAppSessionContextToTrafficInfluSub(rspBody.(*models.AppSessionContext))
			// Not Complete: Need to advise the Self IE
			tiSub.Self = genTrafficInfluSubURI(p.Config().SbiUri(), afID, appSessID)
			tiSubList = append(tiSubList, *tiSub)
		}
	}
	if len(subInUDR) > 0 {
		rspCode, rspBody := p.Consumer().AppDataInfluenceDataGet(subInUDR)
		if rspCode != http.StatusOK {
			return &HandlerResponse{rspCode, nil, rspBody}
		}

		for _, tiData := range *rspBody.(*[]models.TrafficInfluData) {
			tiSub := convertTrafficInfluDataToTrafficInfluSub(&tiData)
			// Not Complete: Need to advise the Self IE
			tiSub.Self = genTrafficInfluSubURI(p.Config().SbiUri(), afID, "0")
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

	afCtx := p.Context().NewAfCtx(afID)
	afSubsc := p.Context().NewAfSubsc(afCtx)
	if len(tiSub.Gpsi) > 0 || len(tiSub.Ipv4Addr) > 0 || len(tiSub.Ipv6Addr) > 0 {
		// Single UE, sent to PCF
		rsp = p.pcfPostAppSessions(afSubsc, tiSub)
	} else if len(tiSub.ExternalGroupId) > 0 || tiSub.AnyUeInd {
		// Group or any UE, sent to UDR
		influenceID := uuid.New().String()
		afSubsc.SetInfluenceID(influenceID)
		tiData := convertTrafficInfluSubToTrafficInfluData(tiSub)
		rspCode, rspBody := p.Consumer().AppDataInfluenceDataPut(influenceID, tiData)
		rsp = &HandlerResponse{
			Status: rspCode,
			Body:   rspBody,
		}
	} else {
		// Invalid case. Return Error
		pd := openapi.ProblemDetailsMalformedReqSyntax("Not individual UE case, nor group case")
		rsp = &HandlerResponse{
			Status: int(pd.Status),
			Body:   pd,
		}
	}

	if rsp.Status >= http.StatusOK && rsp.Status <= http.StatusAlreadyReported {
		p.Context().AddAfCtx(afCtx)
		afCtx.AddSubsc(afSubsc)

		// Create Location URI
		locUri := p.Config().SbiUri() + factory.TraffInfluResUriPrefix + "/" + afID +
			"/subscriptions/" + afSubsc.GetSubscID()
		tiSub.Self = locUri
		rsp.Headers = map[string][]string{
			"Location": {locUri},
		}
	}
	return &HandlerResponse{rsp.Status, rsp.Headers, tiSub}
}

func (p *Processor) GetIndividualTrafficInfluenceSubscription(afID, subscID string) *HandlerResponse {
	logger.TrafInfluLog.Infof("GetIndividualTrafficInfluenceSubscription - afID[%s], subscID[%s]", afID, subscID)

	afCtx := p.Context().GetAfCtx(afID)
	if afCtx == nil {
		problemDetails := openapi.ProblemDetailsDataNotFound("Target AF is not existed")
		return &HandlerResponse{http.StatusNotFound, nil, problemDetails}
	}

	subsc := afCtx.GetSubsc(subscID)
	if afCtx == nil {
		problemDetails := openapi.ProblemDetailsDataNotFound("Target subscription is not existed")
		return &HandlerResponse{http.StatusNotFound, nil, problemDetails}
	}

	if subsc.IsIndividualUEAddr() {
		rspCode, rspBody := p.Consumer().GetAppSession(subsc.GetAppSessID())
		if rspCode != http.StatusOK {
			return &HandlerResponse{rspCode, nil, rspBody}
		}
		tiSub := convertAppSessionContextToTrafficInfluSub(rspBody.(*models.AppSessionContext))
		tiSub.Self = genTrafficInfluSubURI(p.Config().SbiUri(), afID, subscID)
		return &HandlerResponse{http.StatusOK, nil, tiSub}
	} else {
		rspCode, rspBody := p.Consumer().AppDataInfluenceDataIdGet(subsc.GetInfluenceID())
		if rspCode != http.StatusOK {
			return &HandlerResponse{rspCode, nil, rspBody}
		}
		tiSub := convertTrafficInfluDataToTrafficInfluSub(rspBody.(*models.TrafficInfluData))
		tiSub.Self = genTrafficInfluSubURI(p.Config().SbiUri(), afID, subscID)
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

	afCtx := p.Context().GetAfCtx(afID)
	if afCtx == nil {
		problemDetails := openapi.ProblemDetailsDataNotFound("Target AF is not existed")
		return &HandlerResponse{http.StatusNotFound, nil, problemDetails}
	}

	subsc := afCtx.GetSubsc(subscID)
	if afCtx == nil {
		problemDetails := openapi.ProblemDetailsDataNotFound("Target subscription is not existed")
		return &HandlerResponse{http.StatusNotFound, nil, problemDetails}
	}

	if subsc.IsIndividualUEAddr() {
		ascUpdateData := convertTrafficInfluSubPatchToAppSessionContextUpdateData(tiSubPatch)
		rspCode, rspBody := p.Consumer().PatchAppSession(subsc.GetAppSessID(), ascUpdateData)
		if rspCode != http.StatusOK {
			return &HandlerResponse{rspCode, nil, rspBody}
		}
		tiSub := convertAppSessionContextToTrafficInfluSub(rspBody.(*models.AppSessionContext))
		return &HandlerResponse{http.StatusOK, nil, tiSub}
	} else {
		tiDataPatch := convertTrafficInfluSubPatchToTrafficInfluDataPatch(tiSubPatch)
		rspCode, rspBody := p.Consumer().AppDataInfluenceDataPatch(subsc.GetInfluenceID(), tiDataPatch)
		if rspCode != http.StatusOK {
			return &HandlerResponse{rspCode, nil, rspBody}
		}
		tiSub := convertTrafficInfluDataToTrafficInfluSub(rspBody.(*models.TrafficInfluData))
		return &HandlerResponse{http.StatusOK, nil, tiSub}
	}
}

func (p *Processor) DeleteIndividualTrafficInfluenceSubscription(afID, subscID string) *HandlerResponse {
	logger.TrafInfluLog.Infof("DeleteIndividualTrafficInfluenceSubscription - afID[%s], subscID[%s]", afID, subscID)

	afCtx := p.Context().GetAfCtx(afID)
	if afCtx == nil {
		problemDetails := openapi.ProblemDetailsDataNotFound("Target AF is not existed")
		return &HandlerResponse{http.StatusNotFound, nil, problemDetails}
	}

	subsc := afCtx.GetSubsc(subscID)
	if afCtx == nil {
		problemDetails := openapi.ProblemDetailsDataNotFound("Target subscription is not existed")
		return &HandlerResponse{http.StatusNotFound, nil, problemDetails}
	}

	if subsc.IsIndividualUEAddr() {
		rspCode, rspBody := p.Consumer().DeleteAppSession(subsc.GetAppSessID())
		if rspCode != http.StatusOK {
			afCtx.DeleteSubsc(subscID)
			return &HandlerResponse{rspCode, nil, rspBody}
		}
		return &HandlerResponse{http.StatusOK, nil, nil}
	} else {
		rspCode, rspBody := p.Consumer().AppDataInfluenceDataDelete(subsc.GetInfluenceID())
		if rspCode != http.StatusOK {
			afCtx.DeleteSubsc(subscID)
			return &HandlerResponse{rspCode, nil, rspBody}
		}
		return &HandlerResponse{http.StatusOK, nil, nil}
	}
}

func validateTrafficInfluenceData(tiSub *models.TrafficInfluSub) *HandlerResponse {
	if tiSub.AfAppId == "" && len(tiSub.TrafficFilters) == 0 && len(tiSub.EthTrafficFilters) == 0 {
		problemDetails := openapi.
			ProblemDetailsMalformedReqSyntax("One of afAppId, trafficFilters or ethTrafficFilters shall be included")
		return &HandlerResponse{http.StatusNotFound, nil, problemDetails}
	}
	if tiSub.Gpsi == "" && tiSub.Ipv4Addr == "" && tiSub.Ipv6Addr == "" && tiSub.ExternalGroupId == "" &&
		tiSub.AnyUeInd {
		problemDetails := openapi.
			ProblemDetailsMalformedReqSyntax("One of individual UE identifier, External Group Identifier" +
				" or any UE indication anyUeInd shall be included")
		return &HandlerResponse{http.StatusNotFound, nil, problemDetails}
	}
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

	rspCode, rspBody, appSessID := p.Consumer().PostAppSessions(&asc)
	if rspCode == http.StatusCreated {
		afSubsc.SetAppSessID(appSessID)
		return &HandlerResponse{rspCode, nil, nil}
	}
	return &HandlerResponse{rspCode, nil, rspBody}
}

func genTrafficInfluSubURI(sbiURI, afID, subscriptionId string) string {
	// E.g. https://localhost:29505/3gpp-traffic-Influence/v1/{afId}/subscriptions/{subscriptionId}
	return fmt.Sprintf("%s%s/%s/subscriptions/%s",
		sbiURI, factory.TraffInfluResUriPrefix, afID, subscriptionId)
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

func convertTrafficInfluSubPatchToTrafficInfluDataPatch(
	tiSubPatch *models.TrafficInfluSubPatch) *models.TrafficInfluDataPatch {
	tiDataPatch := &models.TrafficInfluDataPatch{}
	return tiDataPatch
}

func convertTrafficInfluSubPatchToAppSessionContextUpdateData(
	tiSubPatch *models.TrafficInfluSubPatch) *models.AppSessionContextUpdateData {
	appSessionCtxUpdate := &models.AppSessionContextUpdateData{}
	return appSessionCtxUpdate
}
