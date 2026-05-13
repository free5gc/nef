package notifier

import (
	"testing"
	"time"

	"github.com/free5gc/openapi/models"
)

func TestFlushNotifications_UnreachableNotifyURI_DoesNotPanic(t *testing.T) {
	notifier, err := NewPfdChangeNotifier()
	if err != nil {
		t.Fatalf("create notifier failed: %v", err)
	}

	notifier.AddPfdSub(&models.PfdSubscription{
		ApplicationIds: []string{"app-nef-dos"},
		NotifyUri:      "http://127.0.0.1:1/notify",
	})

	notifyCtx := notifier.NewPfdNotifyContext()
	notifyCtx.AddNotification("app-nef-dos", &models.PfdChangeNotification{
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
