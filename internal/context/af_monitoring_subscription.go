package context

import (
	"sync"

	"github.com/free5gc/openapi/models"
	"github.com/sirupsen/logrus"
)

type AfMonitoringSubscription struct {
	SubID        string
	MonSub       *models.Nef_MonEvt_MonitoringEventSubscription
	AmfSubID     string // AMF-side (Namf_EventExposure) subscription resource ID, needed to delete it
	NotifCorreID string
	// RemainingReports tracks how many more notifications may be forwarded to the AF.
	// 0 means no per-report limit (only MonitorExpireTime governs lifetime).
	// Protected by the parent AfData.Mu.
	RemainingReports int32
	Log              *logrus.Entry

	// OpMu serializes outbound mutations for this subscription.
	// It must be acquired without holding the parent AfData.Mu.
	OpMu sync.Mutex
}
