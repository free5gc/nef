package notifier

import (
	"context"
	"crypto/x509"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/free5gc/openapi/models"
	"github.com/free5gc/openapi/nef/PFDMgmt"
)

func TestFlushNotifications_UnreachableNotifyURI_DoesNotPanic(t *testing.T) {
	notifier, err := NewPfdChangeNotifier(false)
	if err != nil {
		t.Fatalf("create notifier failed: %v", err)
	}

	notifier.AddPfdSub(&models.Nef_PFDMgmt_PfdSubscription{
		ApplicationIds: []string{"app-nef-dos"},
		NotifyUri:      "http://127.0.0.1:1/notify",
	})

	notifyCtx := notifier.NewPfdNotifyContext()
	notifyCtx.AddNotification("app-nef-dos", &models.Nef_PFDMgmt_PfdChangeNotification{
		ApplicationId: "app-nef-dos",
	})

	notifierPanicked := make(chan interface{}, 1)
	go func() {
		defer func() {
			if p := recover(); p != nil {
				notifierPanicked <- p
			}
			close(notifierPanicked)
		}()
		notifyCtx.FlushNotifications()
	}()

	select {
	case p := <-notifierPanicked:
		if p != nil {
			t.Fatalf("FlushNotifications panicked: %v", p)
		}
	case <-time.After(500 * time.Millisecond):
		// FlushNotifications returns quickly; timeout indicates an unexpected block.
		t.Fatal("FlushNotifications timed out")
	}

	// Allow async notify goroutine to run and fail locally.
	time.Sleep(100 * time.Millisecond)
}

func TestSBIPfdTLSVerification(t *testing.T) {
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	server.StartTLS()
	defer server.Close()
	for _, verify := range []bool{false, true} {
		notifier, err := NewPfdChangeNotifier(verify)
		if err != nil {
			t.Fatal(err)
		}
		notifier.initPfdManagementApiClient()
		req := &PFDMgmt.NnefPFDmanagementNotifyRequest{RequestBody: []models.Nef_PFDMgmt_PfdChangeNotification{{ApplicationId: "tls-test"}}}
		_, err = notifier.clientPfdManagement.PFDSubscriptionsApi.NnefPFDmanagementNotify(context.Background(), server.URL, req)
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
