package context

import (
	"github.com/sirupsen/logrus"

	"bitbucket.org/free5gc-team/openapi/models_nef"
)

type AfSubscription struct {
	SubID              string
	TiSub              *models_nef.TrafficInfluSub
	AppSessID          string // use in single UE case
	InfluID            string // use in multiple UE case
	NotifCorreID       string
	NotificationURI    string
	IsIndividualUEAddr bool // false in UDR, true in PCF
	Log                *logrus.Entry
}
