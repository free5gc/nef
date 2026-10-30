package processor

import (
	"context"
	"crypto/x509"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

func TestPostSmfEventExposureNotificationToAfWithToken(t *testing.T) {
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
	if err := postSmfEventExposureNotificationToAf("http://af.example.com/notify", eeNotif, tokenCtx); err != nil {
		t.Fatalf("post callback failed: %v", err)
	}
}

func TestPostSmfEventExposureNotificationToAfNon2xx(t *testing.T) {
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
	err := postSmfEventExposureNotificationToAf("http://af.example.com/notify", eeNotif, context.TODO())
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
		err := postSmfEventExposureNotificationToAf(
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
