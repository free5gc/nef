package processor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/free5gc/nef/internal/logger"
	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/util/metrics/sbi"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

var afCallbackHTTPClient = &http.Client{}

func (p *Processor) SmfNotification(
	c *gin.Context,
	eeNotif *models.Smf_EvtExpos_NsmfEventExposureNotification,
) {
	logger.TrafInfluLog.Infof("SmfNotification - NotifId[%s]", eeNotif.NotifId)

	af, sub := p.Context().FindAfSub(eeNotif.NotifId)
	if sub == nil {
		pd := openapi.ProblemDetailsDataNotFound("Subscription is not found")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(http.StatusNotFound, pd)
		return
	}

	af.Mu.RLock()
	notifDestination := ""
	if sub.TiSub != nil {
		notifDestination = sub.TiSub.NotificationDestination
	}
	af.Mu.RUnlock()

	if notifDestination == "" {
		pd := openapi.ProblemDetailsSystemFailure("AF notification destination is empty")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(http.StatusInternalServerError, pd)
		return
	}

	afCallbackTokenCtx, pd, err := p.Context().GetTokenCtx(
		models.Nrf_NFMgmt_ServiceName("nnef-callback"), models.Nrf_NFMgmt_NFType_AF)
	if err != nil {
		logger.TrafInfluLog.Errorf("Get token for AF callback failed: %+v", pd)
		failure := openapi.ProblemDetailsSystemFailure("get token for AF callback failed")
		if pd != nil && pd.Cause != "" {
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		} else {
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, failure.Cause)
		}
		c.JSON(http.StatusBadGateway, failure)
		return
	}

	if err := postNotificationToAf(notifDestination, eeNotif, afCallbackTokenCtx); err != nil {
		logger.TrafInfluLog.Errorf("Forward SMF notification to AF failed: %v", err)
		pd := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(http.StatusBadGateway, pd)
		return
	}

	c.Status(http.StatusNoContent)
}

// AmfEventNotification handles a Namf_EventExposure notification and forwards it, translated
// into the AF-facing NefMonitoringNotification shape, to the AF's notificationDestination.
func (p *Processor) AmfEventNotification(
	c *gin.Context,
	notif *models.Amf_EvtExpos_AmfEventNotification,
) {
	logger.SBILog.Infof("AmfEventNotification - NotifyCorrelationId[%s]", notif.NotifyCorrelationId)

	af, monSub := p.Context().FindAfMonSub(notif.NotifyCorrelationId)
	if monSub == nil {
		pd := openapi.ProblemDetailsDataNotFound("Subscription is not found")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(http.StatusNotFound, pd)
		return
	}

	af.Mu.RLock()
	notifDestination := ""
	var monSubCopy models.NefMonitoringEventSubscription
	if monSub.MonSub != nil {
		notifDestination = monSub.MonSub.NotificationDestination
		monSubCopy = *monSub.MonSub
	}
	af.Mu.RUnlock()

	if notifDestination == "" {
		pd := openapi.ProblemDetailsSystemFailure("AF notification destination is empty")
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(http.StatusInternalServerError, pd)
		return
	}

	reports := make([]models.NefMonitoringEventReport, 0, len(notif.ReportList))
	for _, r := range notif.ReportList {
		reports = append(reports, models.NefMonitoringEventReport{
			ExternalId:     monSubCopy.ExternalId,
			Msisdn:         monSubCopy.Msisdn,
			MonitoringType: monSubCopy.MonitoringType,
			EventTime:      r.TimeStamp,
			LocationInfo:   toLocationInfo(r.Location),
			// LossOfConnectReason intentionally left unset: AMF's internal
			// LossOfConnectivityReason has no defined mapping onto the TS 29.336 cause
			// codes this field expects.
		})
	}
	monNotif := &models.NefMonitoringNotification{
		Subscription:           monSubCopy.Self,
		MonitoringEventReports: reports,
	}

	afCallbackTokenCtx, pd, err := p.Context().GetTokenCtx(
		models.Nrf_NFMgmt_ServiceName("nnef-callback"), models.Nrf_NFMgmt_NFType_AF)
	if err != nil {
		logger.SBILog.Errorf("Get token for AF callback failed: %+v", pd)
		failure := openapi.ProblemDetailsSystemFailure("get token for AF callback failed")
		if pd != nil && pd.Cause != "" {
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		} else {
			c.Set(sbi.IN_PB_DETAILS_CTX_STR, failure.Cause)
		}
		c.JSON(http.StatusBadGateway, failure)
		return
	}

	if err := postNotificationToAf(notifDestination, monNotif, afCallbackTokenCtx); err != nil {
		logger.SBILog.Errorf("Forward AMF event notification to AF failed: %v", err)
		pd := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(http.StatusBadGateway, pd)
		return
	}

	c.Status(http.StatusNoContent)
}

// toLocationInfo converts AMF's internal (TS 29.571) UserLocation into the AF-facing
// (TS 29.122) LocationInfo shape used by MonitoringEventReport. Only NR and E-UTRA
// locations are mapped; the other UserLocation variants (N3GA/UTRA/GERA) are not yet
// supported by AMF's LOCATION_REPORT event.
func toLocationInfo(loc *models.UserLocation) *models.NefLocationInfo {
	if loc == nil {
		return nil
	}

	switch {
	case loc.NrLocation != nil:
		nr := loc.NrLocation
		info := &models.NefLocationInfo{
			AgeOfLocationInfo: nr.AgeOfLocationInformation,
		}
		if nr.Ncgi != nil {
			info.CellId = nr.Ncgi.NrCellId
			info.PlmnId = plmnIdString(nr.Ncgi.PlmnId)
		}
		if nr.Tai != nil {
			info.TrackingAreaId = nr.Tai.Tac
			if info.PlmnId == "" {
				info.PlmnId = plmnIdString(nr.Tai.PlmnId)
			}
		}
		return info
	case loc.EutraLocation != nil:
		eutra := loc.EutraLocation
		info := &models.NefLocationInfo{
			AgeOfLocationInfo: eutra.AgeOfLocationInformation,
		}
		if eutra.Ecgi != nil {
			info.EnodeBId = eutra.Ecgi.EutraCellId
			info.PlmnId = plmnIdString(eutra.Ecgi.PlmnId)
		}
		if eutra.Tai != nil {
			info.TrackingAreaId = eutra.Tai.Tac
			if info.PlmnId == "" {
				info.PlmnId = plmnIdString(eutra.Tai.PlmnId)
			}
		}
		return info
	default:
		return nil
	}
}

func plmnIdString(id *models.PlmnId) string {
	if id == nil {
		return ""
	}
	return id.Mcc + "-" + id.Mnc
}

func postNotificationToAf(
	notifDestination string,
	body interface{},
	requestCtx context.Context,
) error {
	contentType, reqBody, err := openapi.Serialize(body, "application/json")
	if err != nil {
		return fmt.Errorf("serialize notification failed: %w", err)
	}
	if requestCtx == nil {
		requestCtx = context.Background()
	}

	httpReq, err := http.NewRequestWithContext(requestCtx, http.MethodPost, notifDestination, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("create AF callback request failed: %w", err)
	}
	httpReq.Header.Set("Content-Type", contentType)
	if err = bindOAuthTokenToRequest(httpReq, requestCtx); err != nil {
		return fmt.Errorf("bind OAuth2 token for AF callback failed: %w", err)
	}

	httpRsp, err := afCallbackHTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("send AF callback failed: %w", err)
	}
	defer func() {
		if _, copyErr := io.Copy(io.Discard, httpRsp.Body); copyErr != nil {
			logger.TrafInfluLog.Warnf("drain AF callback response body failed: %v", copyErr)
		}
		if closeErr := httpRsp.Body.Close(); closeErr != nil {
			logger.TrafInfluLog.Warnf("close AF callback response body failed: %v", closeErr)
		}
	}()

	if httpRsp.StatusCode < http.StatusOK || httpRsp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("AF callback returned status code %d", httpRsp.StatusCode)
	}

	return nil
}

func bindOAuthTokenToRequest(req *http.Request, requestCtx context.Context) error {
	if requestCtx == nil {
		return nil
	}

	tok, ok := requestCtx.Value(openapi.ContextOAuth2).(oauth2.TokenSource)
	if !ok {
		return nil
	}

	latestToken, err := tok.Token()
	if err != nil {
		return err
	}
	latestToken.SetAuthHeader(req)
	return nil
}
