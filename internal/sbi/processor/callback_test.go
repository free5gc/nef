package processor

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestPostSmfEventExposureNotificationToAfOmitsAuthorization(t *testing.T) {
	openapi.InterceptInnerHttp2Client(t, false)
	originalClient := afCallbackHTTPClient
	t.Cleanup(func() { afCallbackHTTPClient = originalClient })

	afCallbackHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.Header.Get("Authorization"); got != "" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}, nil
	})}

	tok := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token-for-af", TokenType: "Bearer"})
	tokenCtx := context.WithValue(context.Background(), openapi.ContextOAuth2, tok)

	eeNotif := &models.Smf_EvtExpos_NsmfEventExposureNotification{NotifId: "notif-1"}
	if err := postNotificationToAf("http://af.example.com/notify", eeNotif, tokenCtx); err != nil {
		t.Fatalf("post callback failed: %v", err)
	}
}

func TestPostNotificationToAfNon2xx(t *testing.T) {
	openapi.InterceptInnerHttp2Client(t, false)
	originalClient := afCallbackHTTPClient
	t.Cleanup(func() { afCallbackHTTPClient = originalClient })

	afCallbackHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusForbidden,
			Body:       io.NopCloser(strings.NewReader("forbidden")),
			Header:     make(http.Header),
		}, nil
	})}

	eeNotif := &models.Smf_EvtExpos_NsmfEventExposureNotification{NotifId: "notif-2"}
	err := postNotificationToAf("http://af.example.com/notify", eeNotif, context.TODO())
	if err == nil {
		t.Fatal("expected error when AF callback returns non-2xx")
	}
}

// newGinContext returns a fresh gin context backed by a ResponseRecorder.
func newGinContext(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", nil)
	return c, w
}

// TestSmfNotification_NotifIdNotFound verifies that SmfNotification returns
// 404 when the NotifId has no matching subscription in NefContext.
func TestSmfNotification_NotifIdNotFound(t *testing.T) {
	openapi.InterceptInnerHttp2Client(t, false)
	c, w := newGinContext(t)

	notif := &models.Smf_EvtExpos_NsmfEventExposureNotification{NotifId: "unknown-corr-id"}
	nefApp.Processor().SmfNotification(c, notif)
	c.Writer.WriteHeaderNow()

	require.Equal(t, http.StatusNotFound, w.Code)
}

// TestSmfNotification_EmptyNotifDest verifies that SmfNotification returns
// 500 when the matching subscription has an empty notificationDestination.
func TestSmfNotification_EmptyNotifDest(t *testing.T) {
	openapi.InterceptInnerHttp2Client(t, false)
	nefCtx := nefApp.Context()
	af := nefCtx.NewAf("af-callback-test-1")
	af.Mu.Lock()
	correID := nefCtx.NewCorreID()
	tiSub := &models.Nef_TrafInfl_TrafficInfluSub{
		NotificationDestination: "", // intentionally empty
	}
	afSub := af.NewSub(correID, tiSub)
	af.Subs[afSub.SubID] = afSub
	nefCtx.AddAf(af)
	af.Mu.Unlock()
	defer func() {
		nefCtx.DeleteAf(af.AfID)
		nefCtx.ResetCorreID()
	}()

	c, w := newGinContext(t)
	notif := &models.Smf_EvtExpos_NsmfEventExposureNotification{NotifId: afSub.NotifCorreID}
	nefApp.Processor().SmfNotification(c, notif)
	c.Writer.WriteHeaderNow()

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestSmfNotification_SuccessfulForward verifies that SmfNotification forwards
// the notification to the AF and returns 204 when AF responds successfully.
func TestSmfNotification_SuccessfulForward(t *testing.T) {
	openapi.InterceptInnerHttp2Client(t, false)
	originalClient := afCallbackHTTPClient
	t.Cleanup(func() { afCallbackHTTPClient = originalClient })

	afReceived := false
	afCallbackHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Empty(t, req.Header.Get("Authorization"))
		afReceived = true
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}, nil
	})}

	nefCtx := nefApp.Context()
	previous := nefCtx.OAuth2Required
	nefCtx.OAuth2Required = true
	t.Cleanup(func() { nefCtx.OAuth2Required = previous })
	af := nefCtx.NewAf("af-callback-test-2")
	af.Mu.Lock()
	correID := nefCtx.NewCorreID()
	tiSub := &models.Nef_TrafInfl_TrafficInfluSub{
		NotificationDestination: "http://af.example.com/notify",
	}
	afSub := af.NewSub(correID, tiSub)
	af.Subs[afSub.SubID] = afSub
	nefCtx.AddAf(af)
	af.Mu.Unlock()
	defer func() {
		nefCtx.DeleteAf(af.AfID)
		nefCtx.ResetCorreID()
	}()

	c, w := newGinContext(t)
	notif := &models.Smf_EvtExpos_NsmfEventExposureNotification{NotifId: afSub.NotifCorreID}
	nefApp.Processor().SmfNotification(c, notif)
	c.Writer.WriteHeaderNow()

	require.Equal(t, http.StatusNoContent, w.Code)
	require.True(t, afReceived, "AF mock should have received the notification")
}

// TestSmfNotification_AfReturnsError verifies that SmfNotification returns
// 502 (BadGateway) when the AF callback endpoint responds with a non-2xx status.
func TestSmfNotification_AfReturnsError(t *testing.T) {
	openapi.InterceptInnerHttp2Client(t, false)
	originalClient := afCallbackHTTPClient
	t.Cleanup(func() { afCallbackHTTPClient = originalClient })

	afCallbackHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusForbidden,
			Body:       io.NopCloser(strings.NewReader(`{"status":403,"title":"Forbidden"}`)),
			Header:     make(http.Header),
		}, nil
	})}

	nefCtx := nefApp.Context()
	af := nefCtx.NewAf("af-callback-test-3")
	af.Mu.Lock()
	correID := nefCtx.NewCorreID()
	tiSub := &models.Nef_TrafInfl_TrafficInfluSub{
		NotificationDestination: "http://af.example.com/notify",
	}
	afSub := af.NewSub(correID, tiSub)
	af.Subs[afSub.SubID] = afSub
	nefCtx.AddAf(af)
	af.Mu.Unlock()
	defer func() {
		nefCtx.DeleteAf(af.AfID)
		nefCtx.ResetCorreID()
	}()

	c, w := newGinContext(t)
	notif := &models.Smf_EvtExpos_NsmfEventExposureNotification{NotifId: afSub.NotifCorreID}
	nefApp.Processor().SmfNotification(c, notif)
	c.Writer.WriteHeaderNow()

	require.Equal(t, http.StatusBadGateway, w.Code)
}

// TestAmfEventNotification_NotifyCorrelationIdNotFound verifies that AmfEventNotification
// returns 404 when the NotifyCorrelationId has no matching monitoring subscription.
func TestAmfEventNotification_NotifyCorrelationIdNotFound(t *testing.T) {
	c, w := newGinContext(t)

	notif := &models.Amf_EvtExpos_AmfEventNotification{NotifyCorrelationId: "unknown-corr-id"}
	nefApp.Processor().AmfEventNotification(c, notif)
	c.Writer.WriteHeaderNow()

	require.Equal(t, http.StatusNotFound, w.Code)
}

// TestAmfEventNotification_EmptyNotifDest verifies that AmfEventNotification returns
// 500 when the matching subscription has an empty notificationDestination.
func TestAmfEventNotification_EmptyNotifDest(t *testing.T) {
	nefCtx := nefApp.Context()
	af := nefCtx.NewAf("af-amf-callback-test-1")
	af.Mu.Lock()
	correID := nefCtx.NewCorreID()
	monSub := &models.Nef_MonEvt_MonitoringEventSubscription{
		NotificationDestination: "", // intentionally empty
	}
	monSubCtx := af.NewMonSub(correID, monSub)
	af.MonSubs[monSubCtx.SubID] = monSubCtx
	nefCtx.AddAf(af)
	af.Mu.Unlock()
	defer func() {
		nefCtx.DeleteAf(af.AfID)
		nefCtx.ResetCorreID()
	}()

	c, w := newGinContext(t)
	notif := &models.Amf_EvtExpos_AmfEventNotification{NotifyCorrelationId: monSubCtx.NotifCorreID}
	nefApp.Processor().AmfEventNotification(c, notif)
	c.Writer.WriteHeaderNow()

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestAmfEventNotification_SuccessfulForward verifies that AmfEventNotification translates
// AMF's report into the AF-facing NefMonitoringNotification shape and forwards it, returning
// 204 when the AF responds successfully.
func TestAmfEventNotification_SuccessfulForward(t *testing.T) {
	originalClient := afCallbackHTTPClient
	t.Cleanup(func() { afCallbackHTTPClient = originalClient })

	var afReceivedBody []byte
	afCallbackHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		// TS 29.500 §6.7.3: notifications are sent without an OAuth2 access token.
		require.Empty(t, req.Header.Get("Authorization"))
		var readErr error
		afReceivedBody, readErr = io.ReadAll(req.Body)
		require.NoError(t, readErr)
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}, nil
	})}

	nefCtx := nefApp.Context()
	af := nefCtx.NewAf("af-amf-callback-test-2")
	af.Mu.Lock()
	correID := nefCtx.NewCorreID()
	monSub := &models.Nef_MonEvt_MonitoringEventSubscription{
		Msisdn:                  "0900000001",
		NotificationDestination: "http://af.example.com/notify",
		MonitoringType:          models.Nef_MonEvt_MonitoringType_UE_REACHABILITY,
		Self:                    "http://nef.example.com/3gpp-monitoring-event/v1/af-amf-callback-test-2/subscriptions/1",
	}
	monSubCtx := af.NewMonSub(correID, monSub)
	af.MonSubs[monSubCtx.SubID] = monSubCtx
	nefCtx.AddAf(af)
	af.Mu.Unlock()
	defer func() {
		nefCtx.DeleteAf(af.AfID)
		nefCtx.ResetCorreID()
	}()

	eventTime := time.Now().UTC()
	notif := &models.Amf_EvtExpos_AmfEventNotification{
		NotifyCorrelationId: monSubCtx.NotifCorreID,
		ReportList: []models.Amf_EvtExpos_AmfEventReport{
			{
				Type:      models.Amf_EvtExpos_AmfEventType_REACHABILITY_REPORT,
				TimeStamp: &eventTime,
			},
		},
	}

	c, w := newGinContext(t)
	nefApp.Processor().AmfEventNotification(c, notif)
	c.Writer.WriteHeaderNow()

	require.Equal(t, http.StatusNoContent, w.Code)
	require.NotEmpty(t, afReceivedBody, "AF mock should have received the notification body")

	var forwarded models.Nef_MonEvt_MonitoringNotification
	require.NoError(t, json.Unmarshal(afReceivedBody, &forwarded))
	require.Equal(t, monSub.Self, forwarded.Subscription)
	require.Len(t, forwarded.MonitoringEventReports, 1)
	require.Equal(t, monSub.Msisdn, forwarded.MonitoringEventReports[0].Msisdn)
	require.Equal(t, monSub.MonitoringType, forwarded.MonitoringEventReports[0].MonitoringType)
}

// TestMonitoringEventNotificationE2E verifies the full notification path without
// seeding state directly: subscribe via PostMonitoringEventSubscription, then
// fire an AMF event notification and assert the AF receives the translated body.
func TestMonitoringEventNotificationE2E(t *testing.T) {
	openapi.InterceptInnerHttp2Client(t, false)
	originalClient := afCallbackHTTPClient
	t.Cleanup(func() { afCallbackHTTPClient = originalClient })

	var afReceivedBody []byte
	afCallbackHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		var err error
		afReceivedBody, err = io.ReadAll(req.Body)
		require.NoError(t, err)
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}, nil
	})}

	initNRFDiscUDMSdmStub()
	initUDMSdmGetSupiOrGpsiStub(http.StatusOK, "imsi-208930000000001")
	initNRFDiscAMFStub()
	initAMFEvtsCreateSubscriptionStub(http.StatusCreated, "amf-sub-e2e")

	sub := models.Nef_MonEvt_MonitoringEventSubscription{
		ExternalId:              "10001@domain.com",
		NotificationDestination: "http://127.0.0.100:8000/notify",
		MonitoringType:          models.Nef_MonEvt_MonitoringType_UE_REACHABILITY,
		MaximumNumberOfReports:  1,
	}

	ginCtx, rec := newGinContext(t)
	nefApp.Processor().PostMonitoringEventSubscription(ginCtx, "af-e2e", &sub)
	ginCtx.Writer.WriteHeaderNow()
	require.Equal(t, http.StatusCreated, rec.Code)

	nefCtx := nefApp.Context()
	t.Cleanup(func() {
		nefCtx.DeleteAf("af-e2e")
		nefCtx.ResetCorreID()
	})

	af := nefCtx.GetAf("af-e2e")
	require.NotNil(t, af)
	af.Mu.RLock()
	var correID string
	for _, monSubCtx := range af.MonSubs {
		correID = monSubCtx.NotifCorreID
		break
	}
	af.Mu.RUnlock()
	require.NotEmpty(t, correID, "subscription must have a correlation ID after create")

	eventTime := time.Now().UTC()
	notif := &models.Amf_EvtExpos_AmfEventNotification{
		NotifyCorrelationId: correID,
		ReportList: []models.Amf_EvtExpos_AmfEventReport{
			{
				Type:      models.Amf_EvtExpos_AmfEventType_REACHABILITY_REPORT,
				TimeStamp: &eventTime,
			},
		},
	}

	ginCtx2, rec2 := newGinContext(t)
	nefApp.Processor().AmfEventNotification(ginCtx2, notif)
	ginCtx2.Writer.WriteHeaderNow()
	require.Equal(t, http.StatusNoContent, rec2.Code)

	require.NotEmpty(t, afReceivedBody, "AF must receive the forwarded notification")
	var forwarded models.Nef_MonEvt_MonitoringNotification
	require.NoError(t, json.Unmarshal(afReceivedBody, &forwarded))
	require.Len(t, forwarded.MonitoringEventReports, 1)
	require.Equal(t, models.Nef_MonEvt_MonitoringType_UE_REACHABILITY, forwarded.MonitoringEventReports[0].MonitoringType)
}

// TestAmfEventNotification_AfReturnsError verifies that AmfEventNotification returns
// 502 (BadGateway) when the AF callback endpoint responds with a non-2xx status.
func TestAmfEventNotification_AfReturnsError(t *testing.T) {
	originalClient := afCallbackHTTPClient
	t.Cleanup(func() { afCallbackHTTPClient = originalClient })

	afCallbackHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusForbidden,
			Body:       io.NopCloser(strings.NewReader(`{"status":403,"title":"Forbidden"}`)),
			Header:     make(http.Header),
		}, nil
	})}

	nefCtx := nefApp.Context()
	af := nefCtx.NewAf("af-amf-callback-test-3")
	af.Mu.Lock()
	correID := nefCtx.NewCorreID()
	monSub := &models.Nef_MonEvt_MonitoringEventSubscription{
		NotificationDestination: "http://af.example.com/notify",
		MonitoringType:          models.Nef_MonEvt_MonitoringType_UE_REACHABILITY,
	}
	monSubCtx := af.NewMonSub(correID, monSub)
	af.MonSubs[monSubCtx.SubID] = monSubCtx
	nefCtx.AddAf(af)
	af.Mu.Unlock()
	defer func() {
		nefCtx.DeleteAf(af.AfID)
		nefCtx.ResetCorreID()
	}()

	c, w := newGinContext(t)
	notif := &models.Amf_EvtExpos_AmfEventNotification{NotifyCorrelationId: monSubCtx.NotifCorreID}
	nefApp.Processor().AmfEventNotification(c, notif)
	c.Writer.WriteHeaderNow()

	require.Equal(t, http.StatusBadGateway, w.Code)
}

// TestAmfEventNotification_MaxReportsEnforced verifies that once MaximumNumberOfReports
// is reached the subscription is removed and a second notification returns 404.
func TestAmfEventNotification_MaxReportsEnforced(t *testing.T) {
	openapi.InterceptInnerHttp2Client(t, false)
	initNRFDiscAMFStub()
	initAMFEvtsDeleteSubscriptionStub(http.StatusNoContent)

	originalClient := afCallbackHTTPClient
	t.Cleanup(func() { afCallbackHTTPClient = originalClient })

	forwardCount := 0
	afCallbackHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		forwardCount++
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}, nil
	})}

	nefCtx := nefApp.Context()
	af := nefCtx.NewAf("af-maxreports-test")
	af.Mu.Lock()
	correID := nefCtx.NewCorreID()
	monSub := &models.Nef_MonEvt_MonitoringEventSubscription{
		NotificationDestination: "http://af.example.com/notify",
		MonitoringType:          models.Nef_MonEvt_MonitoringType_UE_REACHABILITY,
		MaximumNumberOfReports:  1,
	}
	monSubCtx := af.NewMonSub(correID, monSub)
	monSubCtx.AmfSubID = "amf-sub-maxreports"
	af.MonSubs[monSubCtx.SubID] = monSubCtx
	nefCtx.AddAf(af)
	af.Mu.Unlock()
	t.Cleanup(func() {
		nefCtx.DeleteAf(af.AfID)
		nefCtx.ResetCorreID()
	})

	notif := &models.Amf_EvtExpos_AmfEventNotification{NotifyCorrelationId: monSubCtx.NotifCorreID}

	c1, w1 := newGinContext(t)
	nefApp.Processor().AmfEventNotification(c1, notif)
	c1.Writer.WriteHeaderNow()
	require.Equal(t, http.StatusNoContent, w1.Code)

	c2, w2 := newGinContext(t)
	nefApp.Processor().AmfEventNotification(c2, notif)
	c2.Writer.WriteHeaderNow()
	require.Equal(t, http.StatusNotFound, w2.Code)

	require.Equal(t, 1, forwardCount, "AF must receive exactly one notification")
}

// TestAmfEventNotification_ExpiredSubscription verifies that a notification arriving after
// MonitorExpireTime is silently dropped (204), the subscription is removed, and the AMF
// subscription is deleted.
func TestAmfEventNotification_ExpiredSubscription(t *testing.T) {
	openapi.InterceptInnerHttp2Client(t, false)
	initNRFDiscAMFStub()
	initAMFEvtsDeleteSubscriptionStub(http.StatusNoContent)

	originalClient := afCallbackHTTPClient
	t.Cleanup(func() { afCallbackHTTPClient = originalClient })

	afCalled := false
	afCallbackHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		afCalled = true
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}, nil
	})}

	nefCtx := nefApp.Context()
	af := nefCtx.NewAf("af-expiry-test")
	af.Mu.Lock()
	correID := nefCtx.NewCorreID()
	pastTime := time.Now().UTC().Add(-time.Minute)
	monSub := &models.Nef_MonEvt_MonitoringEventSubscription{
		NotificationDestination: "http://af.example.com/notify",
		MonitoringType:          models.Nef_MonEvt_MonitoringType_UE_REACHABILITY,
		MonitorExpireTime:       &pastTime,
	}
	monSubCtx := af.NewMonSub(correID, monSub)
	monSubCtx.AmfSubID = "amf-sub-expiry"
	af.MonSubs[monSubCtx.SubID] = monSubCtx
	nefCtx.AddAf(af)
	af.Mu.Unlock()
	t.Cleanup(func() {
		nefCtx.DeleteAf(af.AfID)
		nefCtx.ResetCorreID()
	})

	c, w := newGinContext(t)
	nefApp.Processor().AmfEventNotification(c, &models.Amf_EvtExpos_AmfEventNotification{
		NotifyCorrelationId: monSubCtx.NotifCorreID,
	})
	c.Writer.WriteHeaderNow()

	require.Equal(t, http.StatusNoContent, w.Code)
	require.False(t, afCalled, "AF must not be called for an expired subscription")

	af.Mu.RLock()
	_, stillExists := af.MonSubs[monSubCtx.SubID]
	af.Mu.RUnlock()
	require.False(t, stillExists, "expired subscription must be removed from the map")
}

func TestSBICallbackTLSVerification(t *testing.T) {
	previous := afCallbackHTTPClient
	t.Cleanup(func() { afCallbackHTTPClient = previous })
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	server.StartTLS()
	defer server.Close()
	for _, verify := range []bool{false, true} {
		configureAFCallbackHTTPClient(verify)
		err := postNotificationToAf(
			server.URL, &models.Smf_EvtExpos_NsmfEventExposureNotification{}, context.Background(),
		)
		if verify {
			var unknown x509.UnknownAuthorityError
			if !errors.As(err, &unknown) {
				t.Fatalf("expected certificate rejection, got %v", err)
			}
		} else if err != nil {
			t.Fatal(err)
		}
	}
}
