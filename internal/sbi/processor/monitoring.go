package processor

import (
	"net/http"
	"net/url"

	"github.com/free5gc/nef/internal/logger"
	"github.com/free5gc/nef/pkg/factory"
	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/util/metrics/sbi"
	"github.com/gin-gonic/gin"
)

// supportedMonitoringTypes are the monitoring types AMF's Namf_EventExposure already exposes.
// Other 3GPP-defined MonitoringType values (e.g. COMMUNICATION_FAILURE, which needs UDM's
// Nudm_EE) are not yet wired up.
var supportedMonitoringTypes = map[models.Nef_MonEvt_MonitoringType]models.Amf_EvtExpos_AmfEventType{
	models.Nef_MonEvt_MonitoringType_LOSS_OF_CONNECTIVITY: models.Amf_EvtExpos_AmfEventType_LOSS_OF_CONNECTIVITY,
	models.Nef_MonEvt_MonitoringType_UE_REACHABILITY:      models.Amf_EvtExpos_AmfEventType_REACHABILITY_REPORT,
	models.Nef_MonEvt_MonitoringType_LOCATION_REPORTING:   models.Amf_EvtExpos_AmfEventType_LOCATION_REPORT,
}

// GetMonitoringEventSubscriptions Read all monitoring event subscriptions for a given AF
// 3GPP TS 29.122 (reused by TS 29.522 for 5GS) Release 17
func (p *Processor) GetMonitoringEventSubscriptions(
	c *gin.Context,
	afID string,
) {
	logger.SBILog.Infof("GetMonitoringEventSubscriptions - afID[%s]", afID)

	af := p.Context().GetAf(afID)
	if af == nil {
		pd := openapi.ProblemDetailsDataNotFound("AF is not found")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}

	af.Mu.RLock()
	defer af.Mu.RUnlock()

	monSubs := make([]models.Nef_MonEvt_MonitoringEventSubscription, 0, len(af.MonSubs))
	for _, sub := range af.MonSubs {
		monSubs = append(monSubs, *sub.MonSub)
	}
	c.JSON(http.StatusOK, &monSubs)
}

// PostMonitoringEventSubscription Create a new monitoring event subscription
// 3GPP TS 29.122 (reused by TS 29.522 for 5GS) Release 17
func (p *Processor) PostMonitoringEventSubscription(
	c *gin.Context,
	afID string,
	monSub *models.Nef_MonEvt_MonitoringEventSubscription,
) {
	logger.SBILog.Infof("PostMonitoringEventSubscription - afID[%s]", afID)

	amfEventType, problemDetails := validateMonitoringEventData(monSub)
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

	af.Mu.Lock()
	defer af.Mu.Unlock()

	correID := nefCtx.NewCorreID()
	monSubCtx := af.NewMonSub(correID, monSub)

	// AMF's event-subscription lookup only matches by Supi, not Gpsi (it never resolves
	// GPSI itself), so NEF must resolve the AF-facing GPSI to a SUPI first via Nudm_SDM.
	gpsi := buildGpsi(monSub.ExternalId, monSub.Msisdn)
	supi, pd, err := p.Consumer().GetSupiFromGpsi(gpsi)
	switch {
	case pd != nil:
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	case err != nil:
		problemDetails := &models.ProblemDetails{
			Status: http.StatusInternalServerError,
			Detail: "Query to UDM failed",
		}
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	amfSubID, pd, err := p.Consumer().CreateEventSubscription(
		supi, []models.Amf_EvtExpos_AmfEventType{amfEventType}, p.genAmfEventNotifyUri(), monSubCtx.NotifCorreID)
	switch {
	case pd != nil:
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	case err != nil:
		problemDetails := &models.ProblemDetails{
			Status: http.StatusInternalServerError,
			Detail: "Query to AMF failed",
		}
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}
	monSubCtx.AmfSubID = amfSubID

	af.MonSubs[monSubCtx.SubID] = monSubCtx
	af.Log.Infoln("Monitoring event subscription is added")

	nefCtx.AddAf(af)

	monSub.Self = p.genMonitoringEventSubURI(afID, monSubCtx.SubID)
	headers := map[string][]string{
		"Location": {monSub.Self},
	}
	for hdrName, hdrValues := range headers {
		for _, hdrValue := range hdrValues {
			c.Header(hdrName, hdrValue)
		}
	}
	c.JSON(http.StatusCreated, monSub)
}

// GetIndividualMonitoringEventSubscription Read an individual monitoring event subscription
// 3GPP TS 29.122 (reused by TS 29.522 for 5GS) Release 17
func (p *Processor) GetIndividualMonitoringEventSubscription(
	c *gin.Context,
	afID, subID string,
) {
	logger.SBILog.Infof("GetIndividualMonitoringEventSubscription - afID[%s], subID[%s]", afID, subID)

	af := p.Context().GetAf(afID)
	if af == nil {
		pd := openapi.ProblemDetailsDataNotFound("AF is not found")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}

	af.Mu.RLock()
	defer af.Mu.RUnlock()

	monSubCtx, ok := af.MonSubs[subID]
	if !ok {
		pd := openapi.ProblemDetailsDataNotFound("Subscription is not found")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}

	c.JSON(http.StatusOK, monSubCtx.MonSub)
}

// DeleteIndividualMonitoringEventSubscription Delete an individual monitoring event subscription
// 3GPP TS 29.122 (reused by TS 29.522 for 5GS) Release 17
func (p *Processor) DeleteIndividualMonitoringEventSubscription(
	c *gin.Context,
	afID, subID string,
) {
	logger.SBILog.Infof("DeleteIndividualMonitoringEventSubscription - afID[%s], subID[%s]", afID, subID)

	af := p.Context().GetAf(afID)
	if af == nil {
		pd := openapi.ProblemDetailsDataNotFound("AF is not found")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}

	af.Mu.Lock()
	defer af.Mu.Unlock()

	monSubCtx, ok := af.MonSubs[subID]
	if !ok {
		pd := openapi.ProblemDetailsDataNotFound("Subscription is not found")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	}

	pd, err := p.Consumer().DeleteEventSubscription(monSubCtx.AmfSubID)
	switch {
	case pd != nil:
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(int(pd.Status), pd)
		return
	case err != nil:
		problemDetails := &models.ProblemDetails{
			Status: http.StatusInternalServerError,
			Detail: "Query to AMF failed",
		}
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, problemDetails.Cause)
		c.JSON(int(problemDetails.Status), problemDetails)
		return
	}

	delete(af.MonSubs, subID)
	c.Status(http.StatusNoContent)
}

func validateMonitoringEventData(
	monSub *models.Nef_MonEvt_MonitoringEventSubscription,
) (models.Amf_EvtExpos_AmfEventType, *models.ProblemDetails) {
	if monSub.NotificationDestination == "" {
		return "", openapi.ProblemDetailsMalformedReqSyntax("Missing notificationDestination")
	}

	parsedNotificationDestination, err := url.ParseRequestURI(monSub.NotificationDestination)
	if err != nil || parsedNotificationDestination.Scheme == "" || parsedNotificationDestination.Host == "" {
		return "", openapi.ProblemDetailsMalformedReqSyntax("Invalid notificationDestination")
	}

	// TS29.122 allows identifying the target UE by externalId, msisdn, ipv4Addr or ipv6Addr.
	// v1 only supports externalId/msisdn: AMF's Namf_EventExposure subscription is keyed by
	// Gpsi (extid-/msisdn- prefixed), which has no IP-address form, so an IP-only request
	// cannot be forwarded to AMF.
	if monSub.ExternalId == "" && monSub.Msisdn == "" {
		if monSub.Ipv4Addr != "" || monSub.Ipv6Addr != "" {
			return "", openapi.
				ProblemDetailsMalformedReqSyntax(
					"Ipv4Addr/Ipv6Addr-only target identification is not yet supported; provide ExternalId or Msisdn")
		}
		return "", openapi.ProblemDetailsMalformedReqSyntax("Missing one of ExternalId, Msisdn")
	}

	amfEventType, ok := supportedMonitoringTypes[monSub.MonitoringType]
	if !ok {
		return "", openapi.
			ProblemDetailsMalformedReqSyntax("Unsupported or missing monitoringType")
	}

	// TS29.122: at least one of maximumNumberOfReports or monitorExpireTime shall be present.
	if monSub.MaximumNumberOfReports == 0 && monSub.MonitorExpireTime == nil {
		return "", openapi.
			ProblemDetailsMalformedReqSyntax("Missing one of MaximumNumberOfReports, MonitorExpireTime")
	}

	return amfEventType, nil
}

// buildGpsi formats an AF-facing target UE identifier as the GPSI string AMF expects
// (3GPP TS 23.003 clause 19.7.2 / clause 4.9.4 for external ID and MSISDN respectively).
func buildGpsi(externalId, msisdn string) string {
	if externalId != "" {
		return "extid-" + externalId
	}
	return "msisdn-" + msisdn
}

func (p *Processor) genMonitoringEventSubURI(
	afID, subscriptionId string,
) string {
	// E.g. https://localhost:29505/3gpp-monitoring-event/v1/{afId}/subscriptions/{subscriptionId}
	return p.Config().ServiceUri(factory.ServiceNefMonitoringEvent) + "/" + afID + "/subscriptions/" + subscriptionId
}

func (p *Processor) genAmfEventNotifyUri() string {
	return p.Config().ServiceUri(factory.ServiceNefCallback) + "/notification/amf-event"
}
