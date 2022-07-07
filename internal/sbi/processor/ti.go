package processor

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"

	nef_context "bitbucket.org/free5gc-team/nef/internal/context"
	"bitbucket.org/free5gc-team/nef/internal/logger"
	"bitbucket.org/free5gc-team/nef/pkg/factory"
	"bitbucket.org/free5gc-team/openapi"
	"bitbucket.org/free5gc-team/openapi/models"
	"bitbucket.org/free5gc-team/openapi/models_nef"
)

func (p *Processor) GetTrafficInfluenceSubscription(
	afID string,
) *HandlerResponse {
	logger.TrafInfluLog.Infof("GetTrafficInfluenceSubscription - afID[%s]", afID)

	var (
		tiSubList []models_nef.TrafficInfluSub
		subInPCF  []string
		subInUDR  []string
	)

	af := p.Context().GetAf(afID)
	if af == nil {
		pd := openapi.ProblemDetailsDataNotFound("AF is not found")
		return &HandlerResponse{http.StatusNotFound, nil, pd}
	}

	af.Mu.RLock()
	defer af.Mu.RUnlock()

	for _, sub := range af.Subs {
		if sub.IsIndividualUEAddr {
			subInPCF = append(subInPCF, sub.AppSessID)
		} else {
			subInUDR = append(subInUDR, sub.InfluID)
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
			tiSub.Self = p.genTrafficInfluSubURI(afID, appSessID)
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
			tiSub.Self = p.genTrafficInfluSubURI(afID, "0")
			tiSubList = append(tiSubList, *tiSub)
		}
	}

	return &HandlerResponse{http.StatusOK, nil, &tiSubList}
}

func (p *Processor) PostTrafficInfluenceSubscription(
	afID string,
	tiSub *models_nef.TrafficInfluSub,
) *HandlerResponse {
	var rsp *HandlerResponse
	logger.TrafInfluLog.Infof("PostTrafficInfluenceSubscription - afID[%s]", afID)

	rsp = validateTrafficInfluenceData(tiSub)
	if rsp != nil {
		return rsp
	}

	nefCtx := p.Context()
	af := nefCtx.GetAf(afID)
	if af == nil {
		af = nefCtx.NewAf(afID)
		if af == nil {
			pd := openapi.ProblemDetailsSystemFailure("No resource can be allocated")
			return &HandlerResponse{int(pd.Status), nil, pd}
		}
	}

	af.Mu.Lock()
	defer af.Mu.Unlock()

	correID := nefCtx.NewCorreID(af)
	afSub := af.NewSub(correID, tiSub)
	if afSub == nil {
		pd := openapi.ProblemDetailsSystemFailure("No resource can be allocated")
		return &HandlerResponse{int(pd.Status), nil, pd}
	}

	if len(tiSub.Gpsi) > 0 || len(tiSub.Ipv4Addr) > 0 || len(tiSub.Ipv6Addr) > 0 {
		// Single UE, sent to PCF
		rsp = p.pcfPostAppSessions(afSub, tiSub)
	} else if len(tiSub.ExternalGroupId) > 0 || tiSub.AnyUeInd {
		// Group or any UE, sent to UDR
		afSub.InfluID = uuid.New().String()
		tiData := convertTrafficInfluSubToTrafficInfluData(tiSub)
		rspCode, rspBody := p.Consumer().AppDataInfluenceDataPut(afSub.InfluID, tiData)
		rsp = &HandlerResponse{
			Status: rspCode,
			Body:   rspBody,
		}
	} else {
		// Invalid case. Return Error
		pd := openapi.ProblemDetailsMalformedReqSyntax("Not individual UE case, nor group case")
		return &HandlerResponse{
			Status: int(pd.Status),
			Body:   pd,
		}
	}

	af.Subs[afSub.SubID] = afSub
	af.Log.Infoln("Subscription is added")

	nefCtx.AddAf(af)

	// Create Location URI
	locUri := p.Config().ServiceUri(factory.ServiceTraffInflu) + "/" + afID +
		"/subscriptions/" + afSub.SubID
	tiSub.Self = locUri
	rsp.Headers = map[string][]string{
		"Location": {locUri},
	}
	return &HandlerResponse{rsp.Status, rsp.Headers, tiSub}
}

func (p *Processor) GetIndividualTrafficInfluenceSubscription(
	afID, subID string,
) *HandlerResponse {
	logger.TrafInfluLog.Infof("GetIndividualTrafficInfluenceSubscription - afID[%s], subID[%s]", afID, subID)

	af := p.Context().GetAf(afID)
	if af == nil {
		pd := openapi.ProblemDetailsDataNotFound("AF is not found")
		return &HandlerResponse{http.StatusNotFound, nil, pd}
	}

	af.Mu.RLock()
	defer af.Mu.RUnlock()

	sub, ok := af.Subs[subID]
	if !ok {
		pd := openapi.ProblemDetailsDataNotFound("Subscription is not found")
		return &HandlerResponse{http.StatusNotFound, nil, pd}
	}

	if sub.IsIndividualUEAddr {
		rspCode, rspBody := p.Consumer().GetAppSession(sub.AppSessID)
		if rspCode != http.StatusOK {
			return &HandlerResponse{rspCode, nil, rspBody}
		}
		tiSub := convertAppSessionContextToTrafficInfluSub(rspBody.(*models.AppSessionContext))
		tiSub.Self = p.genTrafficInfluSubURI(afID, subID)
		return &HandlerResponse{http.StatusOK, nil, tiSub}
	} else {
		rspCode, rspBody := p.Consumer().AppDataInfluenceDataIdGet(sub.InfluID)
		if rspCode != http.StatusOK {
			return &HandlerResponse{rspCode, nil, rspBody}
		}
		tiSub := convertTrafficInfluDataToTrafficInfluSub(rspBody.(*models.TrafficInfluData))
		tiSub.Self = p.genTrafficInfluSubURI(afID, subID)
		return &HandlerResponse{http.StatusOK, nil, tiSub}
	}
}

func (p *Processor) PutIndividualTrafficInfluenceSubscription(
	afID, subID string,
	tiSub *models_nef.TrafficInfluSub,
) *HandlerResponse {
	logger.TrafInfluLog.Infof("PutIndividualTrafficInfluenceSubscription - afID[%s], subID[%s]", afID, subID)
	return &HandlerResponse{http.StatusOK, nil, nil}
}

func (p *Processor) PatchIndividualTrafficInfluenceSubscription(
	afID, subID string,
	tiSubPatch *models_nef.TrafficInfluSubPatch,
) *HandlerResponse {
	logger.TrafInfluLog.Infof("PatchIndividualTrafficInfluenceSubscription - afID[%s], subID[%s]", afID, subID)

	af := p.Context().GetAf(afID)
	if af == nil {
		pd := openapi.ProblemDetailsDataNotFound("AF is not found")
		return &HandlerResponse{http.StatusNotFound, nil, pd}
	}

	af.Mu.Lock()
	defer af.Mu.Unlock()

	sub, ok := af.Subs[subID]
	if !ok {
		pd := openapi.ProblemDetailsDataNotFound("Subscription is not found")
		return &HandlerResponse{http.StatusNotFound, nil, pd}
	}

	if sub.IsIndividualUEAddr {
		ascUpdateData := convertTrafficInfluSubPatchToAppSessionContextUpdateData(tiSubPatch)
		rspCode, rspBody := p.Consumer().PatchAppSession(sub.AppSessID, ascUpdateData)
		if rspCode != http.StatusOK {
			return &HandlerResponse{rspCode, nil, rspBody}
		}
		tiSub := convertAppSessionContextToTrafficInfluSub(rspBody.(*models.AppSessionContext))
		return &HandlerResponse{http.StatusOK, nil, tiSub}
	} else {
		tiDataPatch := convertTrafficInfluSubPatchToTrafficInfluDataPatch(tiSubPatch)
		rspCode, rspBody := p.Consumer().AppDataInfluenceDataPatch(sub.InfluID, tiDataPatch)
		if rspCode != http.StatusOK {
			return &HandlerResponse{rspCode, nil, rspBody}
		}
		tiSub := convertTrafficInfluDataToTrafficInfluSub(rspBody.(*models.TrafficInfluData))
		return &HandlerResponse{http.StatusOK, nil, tiSub}
	}
}

func (p *Processor) DeleteIndividualTrafficInfluenceSubscription(
	afID, subID string,
) *HandlerResponse {
	logger.TrafInfluLog.Infof("DeleteIndividualTrafficInfluenceSubscription - afID[%s], subID[%s]", afID, subID)

	af := p.Context().GetAf(afID)
	if af == nil {
		pd := openapi.ProblemDetailsDataNotFound("AF is not found")
		return &HandlerResponse{http.StatusNotFound, nil, pd}
	}

	af.Mu.Lock()
	defer af.Mu.Unlock()

	sub, ok := af.Subs[subID]
	if !ok {
		pd := openapi.ProblemDetailsDataNotFound("Subscription is not found")
		return &HandlerResponse{http.StatusNotFound, nil, pd}
	}

	if sub.IsIndividualUEAddr {
		rspCode, rspBody := p.Consumer().DeleteAppSession(sub.AppSessID)
		if rspCode != http.StatusOK {
			delete(af.Subs, subID)
			return &HandlerResponse{rspCode, nil, rspBody}
		}
		return &HandlerResponse{http.StatusOK, nil, nil}
	} else {
		rspCode, rspBody := p.Consumer().AppDataInfluenceDataDelete(sub.InfluID)
		if rspCode != http.StatusOK {
			delete(af.Subs, subID)
			return &HandlerResponse{rspCode, nil, rspBody}
		}
		return &HandlerResponse{http.StatusOK, nil, nil}
	}
}

func validateTrafficInfluenceData(
	tiSub *models_nef.TrafficInfluSub,
) *HandlerResponse {
	if tiSub.AfTransId == "" {
		pd := openapi.
			ProblemDetailsMalformedReqSyntax("Missing AfTransID")
		return &HandlerResponse{int(pd.Status), nil, pd}
	}

	// In case AfServiceID  is not present then DNN has to be included in TI
	if tiSub.AfServiceId == "" && tiSub.Dnn == "" {
		pd := openapi.
			ProblemDetailsMalformedReqSyntax("Missing AfServiceId or Dnn")
		return &HandlerResponse{int(pd.Status), nil, pd}
	}

	if tiSub.AfAppId == "" &&
		len(tiSub.TrafficFilters) == 0 &&
		len(tiSub.EthTrafficFilters) == 0 {
		pd := openapi.
			ProblemDetailsMalformedReqSyntax(
				"Missing one of afAppId, trafficFilters or ethTrafficFilters")
		return &HandlerResponse{int(pd.Status), nil, pd}
	}

	if tiSub.Gpsi == "" &&
		tiSub.Ipv4Addr == "" &&
		tiSub.Ipv6Addr == "" &&
		tiSub.ExternalGroupId == "" &&
		!tiSub.AnyUeInd {
		pd := openapi.
			ProblemDetailsMalformedReqSyntax(
				"Missing one of Gpsi, Ipv4Addr, Ipv6Addr, ExternalGroupId, AnyUeInd")
		return &HandlerResponse{int(pd.Status), nil, pd}
	}
	return nil
}

func (p *Processor) pcfPostAppSessions(
	afSub *nef_context.AfSubscription,
	tiSub *models_nef.TrafficInfluSub,
) *HandlerResponse {
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
		afSub.AppSessID = appSessID
		return &HandlerResponse{rspCode, nil, nil}
	}
	return &HandlerResponse{rspCode, nil, rspBody}
}

func (p *Processor) genTrafficInfluSubURI(
	afID, subscriptionId string,
) string {
	// E.g. https://localhost:29505/3gpp-traffic-Influence/v1/{afId}/subscriptions/{subscriptionId}
	return fmt.Sprintf("%s/%s/subscriptions/%s",
		p.Config().ServiceUri(factory.ServiceTraffInflu), afID, subscriptionId)
}

func convertTrafficInfluDataToTrafficInfluSub(
	tiData *models.TrafficInfluData,
) *models_nef.TrafficInfluSub {
	tiSub := &models_nef.TrafficInfluSub{
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

func convertTrafficInfluSubToTrafficInfluData(
	tiSub *models_nef.TrafficInfluSub,
) *models.TrafficInfluData {
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

func convertAppSessionContextToTrafficInfluSub(
	appSessionCtx *models.AppSessionContext,
) *models_nef.TrafficInfluSub {
	tiSub := &models_nef.TrafficInfluSub{
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
	tiSubPatch *models_nef.TrafficInfluSubPatch,
) *models.TrafficInfluDataPatch {
	tiDataPatch := &models.TrafficInfluDataPatch{}
	return tiDataPatch
}

func convertTrafficInfluSubPatchToAppSessionContextUpdateData(
	tiSubPatch *models_nef.TrafficInfluSubPatch,
) *models.AppSessionContextUpdateData {
	appSessionCtxUpdate := &models.AppSessionContextUpdateData{}
	return appSessionCtxUpdate
}
