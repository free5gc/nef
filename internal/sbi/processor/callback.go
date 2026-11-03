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

	if err := postSmfEventExposureNotificationToAf(notifDestination, eeNotif, c.Request.Context()); err != nil {
		logger.TrafInfluLog.Errorf("Forward SMF notification to AF failed: %v", err)
		pd := openapi.ProblemDetailsSystemFailure(err.Error())
		c.Set(sbi.IN_PB_DETAILS_CTX_STR, pd.Cause)
		c.JSON(http.StatusBadGateway, pd)
		return
	}

	c.Status(http.StatusNoContent)
}

func postSmfEventExposureNotificationToAf(
	notifDestination string,
	eeNotif *models.Smf_EvtExpos_NsmfEventExposureNotification,
	requestCtx context.Context,
) error {
	_, reqBody, err := openapi.Serialize(eeNotif, "application/json")
	if err != nil {
		return fmt.Errorf("serialize SMF notification failed: %w", err)
	}
	if requestCtx == nil {
		requestCtx = context.Background()
	}

	httpReq, err := http.NewRequestWithContext(requestCtx, http.MethodPost, notifDestination, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("create AF callback request failed: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
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
