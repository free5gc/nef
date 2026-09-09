package context

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/free5gc/nef/internal/logger"
	"github.com/free5gc/openapi/models"
	"github.com/sirupsen/logrus"
)

type AfData struct {
	AfID          string
	NumSubscID    uint64
	NumTransID    uint64
	NumMonSubscID uint64
	Subs          map[string]*AfSubscription
	PfdTrans      map[string]*AfPfdTransaction
	MonSubs       map[string]*AfMonitoringSubscription
	Mu            sync.RWMutex
	Log           *logrus.Entry
}

func (a *AfData) NewSub(numCorreID uint64, tiSub *models.Nef_TrafInfl_TrafficInfluSub) *AfSubscription {
	a.NumSubscID++
	sub := AfSubscription{
		NotifCorreID: strconv.FormatUint(numCorreID, 10),
		SubID:        strconv.FormatUint(a.NumSubscID, 10),
		TiSub:        tiSub,
		Log:          a.Log.WithField(logger.FieldSubID, fmt.Sprintf("SUB:%d", a.NumSubscID)),
	}
	sub.Log.Infoln("New subscription")
	return &sub
}

func (a *AfData) NewMonSub(
	numCorreID uint64, monSub *models.Nef_MonEvt_MonitoringEventSubscription,
) *AfMonitoringSubscription {
	a.NumMonSubscID++
	sub := AfMonitoringSubscription{
		NotifCorreID: strconv.FormatUint(numCorreID, 10),
		SubID:        strconv.FormatUint(a.NumMonSubscID, 10),
		MonSub:       monSub,
		Log:          a.Log.WithField(logger.FieldSubID, fmt.Sprintf("MONSUB:%d", a.NumMonSubscID)),
	}
	sub.Log.Infoln("New monitoring event subscription")
	return &sub
}

func (a *AfData) NewPfdTrans() *AfPfdTransaction {
	a.NumTransID++
	pfdTr := AfPfdTransaction{
		TransID:   strconv.FormatUint(a.NumTransID, 10),
		ExtAppIDs: make(map[string]struct{}),
		Log:       a.Log.WithField(logger.FieldPfdTransID, fmt.Sprintf("PFDT:%d", a.NumTransID)),
	}
	pfdTr.Log.Infoln("New pfd transcation")
	return &pfdTr
}

func (a *AfData) IsAppIDExisted(appID string) (string, bool) {
	for _, pfdTrans := range a.PfdTrans {
		if _, ok := pfdTrans.ExtAppIDs[appID]; ok {
			return pfdTrans.TransID, true
		}
	}
	return "", false
}
