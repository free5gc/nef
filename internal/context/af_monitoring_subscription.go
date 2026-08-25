package context

import (
	"github.com/free5gc/openapi/models"
	"github.com/sirupsen/logrus"
)

type AfMonitoringSubscription struct {
	SubID        string
	MonSub       *models.Nef_MonEvt_MonitoringEventSubscription
	AmfSubID     string // AMF-side (Namf_EventExposure) subscription resource ID, needed to delete it
	NotifCorreID string
	Log          *logrus.Entry
}
