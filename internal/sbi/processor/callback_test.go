package processor

import (
	"context"
	"encoding/json"
	"io"
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

func TestBindOAuthTokenToRequest(t *testing.T) {
	openapi.InterceptInnerHttp2Client(t, false)
	t.Run("context without token source", func(t *testing.T) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "http://af.example.com", nil)
		if err != nil {
			t.Fatalf("create request failed: %v", err)
		}

		err = bindOAuthTokenToRequest(req, context.TODO())
		if err != nil {
			t.Fatalf("bind token failed: %v", err)
		}
		if got := req.Header.Get("Authorization"); got != "" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
	})

	t.Run("context with token source", func(t *testing.T) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "http://example.com", nil)
		if err != nil {
			t.Fatalf("create request failed: %v", err)
		}

		tok := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "abc123", TokenType: "Bearer"})
		tokenCtx := context.WithValue(context.Background(), openapi.ContextOAuth2, tok)

		err = bindOAuthTokenToRequest(req, tokenCtx)
		if err != nil {
			t.Fatalf("bind token failed: %v", err)
		}
		if got := req.Header.Get("Authorization"); got != "Bearer abc123" {
			t.Fatalf("authorization header = %q, want %q", got, "Bearer abc123")
		}
	})
}

func TestPostNotificationToAfWithToken(t *testing.T) {
	openapi.InterceptInnerHttp2Client(t, false)
	originalClient := afCallbackHTTPClient
	t.Cleanup(func() { afCallbackHTTPClient = originalClient })

	afCallbackHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.Header.Get("Authorization"); got != "Bearer token-for-af" {
			t.Fatalf("authorization header = %q, want %q", got, "Bearer token-for-af")
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
func newGinContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

// TestSmfNotification_NotifIdNotFound verifies that SmfNotification returns
// 404 when the NotifId has no matching subscription in NefContext.
func TestSmfNotification_NotifIdNotFound(t *testing.T) {
	openapi.InterceptInnerHttp2Client(t, false)
	c, w := newGinContext()

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

	c, w := newGinContext()
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
		afReceived = true
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}, nil
	})}

	nefCtx := nefApp.Context()
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

	c, w := newGinContext()
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

	c, w := newGinContext()
	notif := &models.Smf_EvtExpos_NsmfEventExposureNotification{NotifId: afSub.NotifCorreID}
	nefApp.Processor().SmfNotification(c, notif)
	c.Writer.WriteHeaderNow()

	require.Equal(t, http.StatusBadGateway, w.Code)
}

// TestAmfEventNotification_NotifyCorrelationIdNotFound verifies that AmfEventNotification
// returns 404 when the NotifyCorrelationId has no matching monitoring subscription.
func TestAmfEventNotification_NotifyCorrelationIdNotFound(t *testing.T) {
	c, w := newGinContext()

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

	c, w := newGinContext()
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

	c, w := newGinContext()
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

	c, w := newGinContext()
	notif := &models.Amf_EvtExpos_AmfEventNotification{NotifyCorrelationId: monSubCtx.NotifCorreID}
	nefApp.Processor().AmfEventNotification(c, notif)
	c.Writer.WriteHeaderNow()

	require.Equal(t, http.StatusBadGateway, w.Code)
}
