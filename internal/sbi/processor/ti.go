package processor

import (
	"net/http"
	"net/url"

	nef_context "github.com/free5gc/nef/internal/context"
	"github.com/free5gc/nef/internal/logger"
	"github.com/free5gc/nef/pkg/factory"
	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/util/metrics/sbi"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetTrafficInfluenceSubscription Read all subscriptions for a given AF
// 3GPP TS 29.522 Release 17 version 17.6.0
// Resource structure: 5.4.1
// Request/Response  : 5.4.1.2.3.2
func (p *Processor) GetTrafficInfluenceSubscription(
	c *gin.Context,
	afID string,
) {
	logger.TrafInfluLog.Infof("GetTrafficInfluenceSubscription - afID[%s]", afID)

	af := p.Context().GetAf(afID)
	if af == nil {
		pd := openapi.ProblemDetailsDataNotFound("AF is not found")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(http.StatusNotFound, pd)
		return
	}

	af.Mu.RLock()
	defer af.Mu.RUnlock()

	var tiSubs []models.Nef_TrafInfl_TrafficInfluSub
	for _, sub := range af.Subs {
		if sub.TiSub == nil {
			continue
		}
		tiSubs = append(tiSubs, *sub.TiSub)
	}
	c.JSON(http.StatusOK, &tiSubs)
}

// PostTrafficInfluenceSubscription Create a new subscription to traffic influence
// 3GPP TS 29.522 Release 17 version 17.6.0
// Resource structure: 5.4.1
// Request/Response  : 5.4.1.2.3.3
func (p *Processor) PostTrafficInfluenceSubscription(
	c *gin.Context,
	afID string,
	tiSub *models.Nef_TrafInfl_TrafficInfluSub,
) {
	logger.TrafInfluLog.Infof("PostTrafficInfluenceSubscription - afID[%s]", afID)

	problemDetails := validateTrafficInfluenceData(tiSub)
	if problemDetails != nil {
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	nefCtx := p.Context()
	af := nefCtx.GetAf(afID)
	if af == nil {
		af = nefCtx.NewAf(afID)
		if af == nil {
			pd := openapi.ProblemDetailsSystemFailure("No resource can be allocated")
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
			c.JSON(int(pd.Status), pd)
			return
		}
	}

	correID := nefCtx.NewCorreID()

	// Reserve the subscription ID under af.Mu; outbound calls below run without it.
	af.Mu.Lock()
	afSub := af.NewSub(correID, tiSub)
	af.Mu.Unlock()
	if afSub == nil {
		pd := openapi.ProblemDetailsSystemFailure("No resource can be allocated")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}
	if len(tiSub.Gpsi) > 0 || len(tiSub.Ipv4Addr) > 0 || len(tiSub.Ipv6Addr) > 0 {
		// Single UE, sent to PCF
		asc := p.convertTrafficInfluSubToAppSessionContext(tiSub, afSub.NotifCorreID)
		appSessId, pd, err := p.Consumer().PostAppSessions(asc)
		switch {
		case pd != nil:
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
			c.JSON(int(pd.Status), pd)
			return
		case err != nil:
			problemDetails := &models.ProblemDetails{
				Status: http.StatusInternalServerError,
				Detail: "Query to PCF failed",
			}
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
			c.JSON(int(problemDetails.Status), problemDetails)
			return
		default:
			afSub.AppSessID = appSessId
		}
	} else if len(tiSub.ExternalGroupId) > 0 || tiSub.AnyUeInd {
		// Group or any UE, sent to UDR
		afSub.InfluID = uuid.New().String()
		tiData := p.convertTrafficInfluSubToTrafficInfluData(tiSub, afSub.NotifCorreID)

		_, pd, err := p.Consumer().AppDataInfluenceDataPut(afSub.InfluID, tiData)
		switch {
		case pd != nil:
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
			c.JSON(int(pd.Status), pd)
			return
		case err != nil:
			problemDetails := &models.ProblemDetails{
				Status: http.StatusInternalServerError,
				Detail: "Query to UDR failed",
			}
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
			c.JSON(int(problemDetails.Status), problemDetails)
			return
		}
	} else {
		// Invalid case. Return Error
		pd := openapi.ProblemDetailsMalformedReqSyntax("Not individual UE case, nor group case")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}

	tiSub.Self = p.genTrafficInfluSubURI(afID, afSub.SubID)
	af.Mu.Lock()
	af.Subs[afSub.SubID] = afSub
	af.Mu.Unlock()
	af.Log.Infoln("Subscription is added")

	// Avoid nesting the global context lock inside af.Mu.
	nefCtx.AddAf(af)

	headers := map[string][]string{
		"Location": {tiSub.Self},
	}

	for hdrName, hdrValues := range headers {
		for _, hdrValue := range hdrValues {
			c.Header(hdrName, hdrValue)
		}
	}
	af.Log.Infoln("Convert TI 3")
	c.JSON(http.StatusCreated, tiSub)
}

// GetIndividualTrafficInfluenceSubscription Read a subscription to traffic influence
// 3GPP TS 29.522 Release 17 version 17.6.0
// Resource structure: 5.4.1
// Request/Response  : 5.4.1.3.3.2
func (p *Processor) GetIndividualTrafficInfluenceSubscription(
	c *gin.Context,
	afID, subID string,
) {
	logger.TrafInfluLog.Infof("GetIndividualTrafficInfluenceSubscription - afID[%s], subID[%s]", afID, subID)

	af := p.Context().GetAf(afID)
	if af == nil {
		pd := openapi.ProblemDetailsDataNotFound("AF is not found")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}

	af.Mu.RLock()
	defer af.Mu.RUnlock()

	afSub, ok := af.Subs[subID]
	if !ok {
		pd := openapi.ProblemDetailsDataNotFound("Subscription is not found")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}

	c.JSON(http.StatusOK, afSub.TiSub)
}

// PutIndividualTrafficInfluenceSubscription Modify all the properties of an existing subscription to traffic influence
// 3GPP TS 29.522 Release 17 version 17.6.0
// Resource structure: 5.4.1
// Request/Response  : 5.4.1.3.3.3
func (p *Processor) PutIndividualTrafficInfluenceSubscription(
	c *gin.Context,
	afID, subID string,
	tiSub *models.Nef_TrafInfl_TrafficInfluSub,
) {
	logger.TrafInfluLog.Infof("PutIndividualTrafficInfluenceSubscription - afID[%s], subID[%s]", afID, subID)

	problemDetails := validateTrafficInfluenceData(tiSub)
	if problemDetails != nil {
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	af := p.Context().GetAf(afID)
	if af == nil {
		pd := openapi.ProblemDetailsDataNotFound("AF is not found")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}

	afSub, ok := lockTrafficInfluenceSubscription(af, subID)
	if !ok {
		pd := openapi.ProblemDetailsDataNotFound("Subscription is not found")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}
	defer afSub.OpMu.Unlock()

	af.Mu.RLock()
	appSessID := afSub.AppSessID
	influID := afSub.InfluID
	notifCorreID := afSub.NotifCorreID
	af.Mu.RUnlock()

	updatedAppSessID := appSessID
	if appSessID != "" {
		asc := p.convertTrafficInfluSubToAppSessionContext(tiSub, notifCorreID)
		appSessId, pd, err := p.Consumer().PostAppSessions(asc)

		switch {
		case pd != nil:
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
			c.JSON(int(pd.Status), pd)
			return
		case err != nil:
			problemDetails := &models.ProblemDetails{
				Status: http.StatusInternalServerError,
				Detail: "Query to PCF failed",
			}
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
			c.JSON(int(problemDetails.Status), problemDetails)
			return
		default:
			updatedAppSessID = appSessId
		}
	} else if influID != "" {
		tiData := p.convertTrafficInfluSubToTrafficInfluData(tiSub, notifCorreID)

		_, pd, err := p.Consumer().AppDataInfluenceDataPut(influID, tiData)
		switch {
		case pd != nil:
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
			c.JSON(int(pd.Status), pd)
			return
		case err != nil:
			problemDetails := &models.ProblemDetails{
				Status: http.StatusInternalServerError,
				Detail: "Query to UDR failed",
			}
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
			c.JSON(int(problemDetails.Status), problemDetails)
			return
		}
	} else {
		pd := openapi.ProblemDetailsDataNotFound("No AppSessID or InfluID")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}

	af.Mu.Lock()
	afSub.TiSub = tiSub
	afSub.AppSessID = updatedAppSessID
	af.Mu.Unlock()

	c.JSON(http.StatusOK, afSub.TiSub)
}

// PatchIndividualTrafficInfluenceSubscription Modify part of the properties of an existing subscription
// to traffic influence
// 3GPP TS 29.522 Release 17 version 17.6.0
// Resource structure: 5.4.1
// Request/Response  : 5.4.1.3.3.4
func (p *Processor) PatchIndividualTrafficInfluenceSubscription(
	c *gin.Context,
	afID, subID string,
	tiSubPatch *models.Nef_TrafInfl_TrafficInfluSubPatch,
) {
	logger.TrafInfluLog.Infof("PatchIndividualTrafficInfluenceSubscription - afID[%s], subID[%s]", afID, subID)

	af := p.Context().GetAf(afID)
	if af == nil {
		pd := openapi.ProblemDetailsDataNotFound("AF is not found")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}

	afSub, ok := lockTrafficInfluenceSubscription(af, subID)
	if !ok {
		pd := openapi.ProblemDetailsDataNotFound("Subscription is not found")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}
	defer afSub.OpMu.Unlock()

	af.Mu.RLock()
	appSessID := afSub.AppSessID
	influID := afSub.InfluID
	af.Mu.RUnlock()

	if appSessID != "" {
		ascUpdateData := p.convertTrafficInfluSubPatchToAppSessionContextUpdateData(tiSubPatch)

		_, pd, err := p.Consumer().PatchAppSession(appSessID, ascUpdateData)
		switch {
		case pd != nil:
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
			c.JSON(int(pd.Status), pd)
			return
		case err != nil:
			problemDetails := &models.ProblemDetails{
				Status: http.StatusInternalServerError,
				Detail: "Query to PCF failed",
			}
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
			c.JSON(int(problemDetails.Status), problemDetails)
			return
		}
	} else if influID != "" {
		tiDataPatch := p.convertTrafficInfluSubPatchToTrafficInfluDataPatch(tiSubPatch)
		_, pd, err := p.Consumer().AppDataInfluenceDataPatch(influID, tiDataPatch)

		switch {
		case pd != nil:
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
			c.JSON(int(pd.Status), pd)
			return
		case err != nil:
			problemDetails := &models.ProblemDetails{
				Status: http.StatusInternalServerError,
				Detail: "Query to UDR failed",
			}
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
			c.JSON(int(problemDetails.Status), problemDetails)
			return
		}
	} else {
		pd := openapi.ProblemDetailsDataNotFound("No AppSessID or InfluID")
		c.JSON(int(pd.Status), pd)
		return
	}

	af.Mu.Lock()
	afSub.PatchTiSubData(tiSubPatch)
	af.Mu.Unlock()
	c.JSON(http.StatusOK, afSub.TiSub)
}

// DeleteIndividualTrafficInfluenceSubscription Delete a subscription to traffic influence
// 3GPP TS 29.522 Release 17 version 17.6.0
// Resource structure: 5.4.1
// Request/Response  : 5.4.1.3.3.5
func (p *Processor) DeleteIndividualTrafficInfluenceSubscription(
	c *gin.Context,
	afID, subID string,
) {
	logger.TrafInfluLog.Infof("DeleteIndividualTrafficInfluenceSubscription - afID[%s], subID[%s]", afID, subID)

	af := p.Context().GetAf(afID)
	if af == nil {
		pd := openapi.ProblemDetailsDataNotFound("AF is not found")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}

	sub, ok := lockTrafficInfluenceSubscription(af, subID)
	if !ok {
		pd := openapi.ProblemDetailsDataNotFound("Subscription is not found")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}
	defer sub.OpMu.Unlock()

	af.Mu.RLock()
	appSessID := sub.AppSessID
	influID := sub.InfluID
	af.Mu.RUnlock()

	if appSessID != "" {
		_, pd, err := p.Consumer().DeleteAppSession(appSessID)
		switch {
		case err != nil:
			problemDetails := &models.ProblemDetails{
				Status: http.StatusInternalServerError,
				Detail: "Query to PCF failed",
			}
			c.JSON(int(problemDetails.Status), problemDetails)
			return
		case pd != nil:
			c.JSON(int(pd.Status), pd)
			return
		}
	} else {
		pd, errInfluenceDataDelete := p.Consumer().AppDataInfluenceDataDelete(influID)

		switch {
		case pd != nil:
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
			c.JSON(int(pd.Status), pd)
			return
		case errInfluenceDataDelete != nil:
			problemDetails := &models.ProblemDetails{
				Status: http.StatusInternalServerError,
				Detail: "Query to UDR failed",
			}
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
			c.JSON(int(problemDetails.Status), problemDetails)
			return
		}
	}
	af.Mu.Lock()
	delete(af.Subs, subID)
	af.Mu.Unlock()
	c.Status(http.StatusNoContent)
}

// lockTrafficInfluenceSubscription locks the current subscription for mutation.
// OpMu is acquired outside af.Mu, then map membership is revalidated.
func lockTrafficInfluenceSubscription(
	af *nef_context.AfData,
	subID string,
) (*nef_context.AfSubscription, bool) {
	af.Mu.RLock()
	sub, ok := af.Subs[subID]
	af.Mu.RUnlock()
	if !ok {
		return nil, false
	}

	sub.OpMu.Lock()

	af.Mu.RLock()
	current, ok := af.Subs[subID]
	af.Mu.RUnlock()
	if !ok || current != sub {
		sub.OpMu.Unlock()
		return nil, false
	}

	return sub, true
}

func validateTrafficInfluenceData(
	tiSub *models.Nef_TrafInfl_TrafficInfluSub,
) *models.ProblemDetails {
	if tiSub.NotificationDestination == "" {
		pd := openapi.ProblemDetailsMalformedReqSyntax("Missing notificationDestination")
		return pd
	}

	parsedNotificationDestination, err := url.ParseRequestURI(tiSub.NotificationDestination)
	if err != nil || parsedNotificationDestination.Scheme == "" || parsedNotificationDestination.Host == "" {
		pd := openapi.ProblemDetailsMalformedReqSyntax("Invalid notificationDestination")
		return pd
	}

	// TS29.522: One of "afAppId", "trafficFilters" or "ethTrafficFilters" shall be included.
	if tiSub.AfAppId == "" &&
		len(tiSub.TrafficFilters) == 0 &&
		len(tiSub.EthTrafficFilters) == 0 {
		pd := openapi.
			ProblemDetailsMalformedReqSyntax(
				"Missing one of afAppId, trafficFilters or ethTrafficFilters")
		return pd
	}

	// TS29.522: One of individual UE identifier
	// (i.e. "gpsi", “macAddr”, "ipv4Addr" or "ipv6Addr"),
	// External Group Identifier (i.e. "externalGroupId") or
	// any UE indication "anyUeInd" shall be included.
	if tiSub.Gpsi == "" &&
		tiSub.Ipv4Addr == "" &&
		tiSub.Ipv6Addr == "" &&
		tiSub.ExternalGroupId == "" &&
		!tiSub.AnyUeInd {
		pd := openapi.
			ProblemDetailsMalformedReqSyntax(
				"Missing one of Gpsi, Ipv4Addr, Ipv6Addr, ExternalGroupId, AnyUeInd")
		return pd
	}

	if len(tiSub.TrafficRoutes) == 0 {
		pd := openapi.
			ProblemDetailsMalformedReqSyntax(
				"Missing trafficRoutes")
		return pd
	}

	// TrafficRoutes is a value slice in the new openapi models, so its
	// elements can no longer be nil; the per-element nil check is gone.
	return nil
}

func (p *Processor) genTrafficInfluSubURI(
	afID, subscriptionId string,
) string {
	// E.g. https://localhost:29505/3gpp-traffic-Influence/v1/{afId}/subscriptions/{subscriptionId}
	return p.Config().ServiceUri(factory.ServiceTraffInflu) + "/" + afID + "/subscriptions/" + subscriptionId
}

func (p *Processor) genNotificationUri() string {
	return p.Config().ServiceUri(factory.ServiceNefCallback) + "/notification/smf"
}

func (p *Processor) convertTrafficInfluSubToAppSessionContext(
	tiSub *models.Nef_TrafInfl_TrafficInfluSub,
	notifCorreID string,
) *models.Pcf_PolAuth_AppSessionContext {
	asc := &models.Pcf_PolAuth_AppSessionContext{
		AscReqData: &models.Pcf_PolAuth_AppSessionContextReqData{
			AfAppId: tiSub.AfAppId,
			AfRoutReq: &models.Pcf_PolAuth_AfRoutingRequirement{
				AppReloc:    tiSub.AppReloInd,
				RouteToLocs: tiSub.TrafficRoutes,
				TempVals:    tiSub.TempValidities,
			},
			UeIpv4:    tiSub.Ipv4Addr,
			UeIpv6:    tiSub.Ipv6Addr,
			UeMac:     tiSub.MacAddr,
			NotifUri:  tiSub.NotificationDestination,
			SuppFeat:  tiSub.SuppFeat,
			Dnn:       tiSub.Dnn,
			SliceInfo: tiSub.Snssai,
			// Supi: ,
		},
	}

	if tiSub.DnaiChgType != "" {
		asc.AscReqData.AfRoutReq.UpPathChgSub = &models.Pcf_SMPolCtrl_UpPathChgEvent{
			DnaiChgType:     tiSub.DnaiChgType,
			NotificationUri: p.genNotificationUri(),
			NotifCorreId:    notifCorreID,
		}
	}
	return asc
}

func (p *Processor) convertTrafficInfluSubPatchToAppSessionContextUpdateData(
	tiSubPatch *models.Nef_TrafInfl_TrafficInfluSubPatch,
) *models.Pcf_PolAuth_AppSessionContextUpdateData {
	ascUpdate := &models.Pcf_PolAuth_AppSessionContextUpdateData{
		AfRoutReq: &models.Pcf_PolAuth_AfRoutingRequirementRm{
			AppReloc:    tiSubPatch.AppReloInd,
			RouteToLocs: tiSubPatch.TrafficRoutes,
			TempVals:    tiSubPatch.TempValidities,
		},
	}
	return ascUpdate
}

func (p *Processor) convertTrafficInfluSubToTrafficInfluData(
	tiSub *models.Nef_TrafInfl_TrafficInfluSub,
	notifCorreID string,
) *models.Udr_DR_TrafficInfluData {
	tiData := &models.Udr_DR_TrafficInfluData{
		AfAppId:    tiSub.AfAppId,
		AppReloInd: tiSub.AppReloInd,
		// Supi: ,
		DnaiChgType:           tiSub.DnaiChgType,
		UpPathChgNotifUri:     p.genNotificationUri(),
		UpPathChgNotifCorreId: notifCorreID,
		Dnn:                   tiSub.Dnn,
		Snssai:                tiSub.Snssai,
		EthTrafficFilters:     tiSub.EthTrafficFilters,
		TrafficFilters:        tiSub.TrafficFilters,
		TrafficRoutes:         tiSub.TrafficRoutes,
		TraffCorreInd:         tiSub.TfcCorrInd,
		// ValidStartTime: ,
		// ValidEndTime: ,
		TempValidities:    tiSub.TempValidities,
		AfAckInd:          tiSub.AfAckInd,
		AddrPreserInd:     tiSub.AddrPreserInd,
		SupportedFeatures: tiSub.SuppFeat,
	}

	// TODO: handle ExternalGroupId
	if tiSub.AnyUeInd {
		tiData.InterGroupId = "AnyUE"
	}

	return tiData
}

func (p *Processor) convertTrafficInfluSubPatchToTrafficInfluDataPatch(
	tiSubPatch *models.Nef_TrafInfl_TrafficInfluSubPatch,
) *models.Udr_DR_TrafficInfluDataPatch {
	tiDataPatch := &models.Udr_DR_TrafficInfluDataPatch{
		AppReloInd:        tiSubPatch.AppReloInd,
		EthTrafficFilters: tiSubPatch.EthTrafficFilters,
		TrafficFilters:    tiSubPatch.TrafficFilters,
		TrafficRoutes:     tiSubPatch.TrafficRoutes,
	}
	return tiDataPatch
}
