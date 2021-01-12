package context

import (
	"strconv"
	"sync"

	"bitbucket.org/free5gc-team/nef/internal/logger"
)

type AfContext struct {
	afID       string
	numSubscID uint64
	numTransID uint64
	subsc      map[string]*AfSubscription
	pfdTrans   map[string]*AfPfdTransaction
	mtx        sync.RWMutex
}

func (a *AfContext) GetAfID() string {
	a.mtx.RLock()
	defer a.mtx.RUnlock()
	return a.afID
}

func (a *AfContext) newSubsc(numCorreID uint64) *AfSubscription {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	a.numSubscID++
	afSubsc := AfSubscription{
		notifCorreID: strconv.FormatUint(numCorreID, 10),
		subscID:      strconv.FormatUint(a.numSubscID, 10),
	}
	return &afSubsc
}

func (a *AfContext) AddSubsc(afSubsc *AfSubscription) {
	a.mtx.Lock()
	a.subsc[afSubsc.subscID] = afSubsc
	a.mtx.Unlock()
	logger.CtxLog.Infof("New AF subscription[%s] added", afSubsc.subscID)
}

func (a *AfContext) newPfdTrans() *AfPfdTransaction {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	a.numTransID++
	afPfdTrans := &AfPfdTransaction{
		transID:        strconv.FormatUint(a.numTransID, 10),
		externalAppIDs: make(map[string]bool),
	}
	return afPfdTrans
}

func (a *AfContext) AddPfdTrans(afPfdTrans *AfPfdTransaction) {
	a.mtx.Lock()
	a.pfdTrans[afPfdTrans.transID] = afPfdTrans
	a.mtx.Unlock()
	logger.CtxLog.Infof("New AF PFD transaction[%s] added", afPfdTrans.transID)
}

func (a *AfContext) GetPfdTrans(transID string) *AfPfdTransaction {
	a.mtx.RLock()
	defer a.mtx.RUnlock()
	return a.pfdTrans[transID]
}

func (a *AfContext) GetAllPfdTrans() []*AfPfdTransaction {
	a.mtx.RLock()
	defer a.mtx.RUnlock()

	allPfdTrans := make([]*AfPfdTransaction, 0, len(a.pfdTrans))
	for _, afPfdTran := range a.pfdTrans {
		allPfdTrans = append(allPfdTrans, afPfdTran)
	}
	return allPfdTrans
}

func (a *AfContext) DeletePfdTrans(transID string) {
	a.mtx.Lock()
	delete(a.pfdTrans, transID)
	a.mtx.Unlock()
	logger.CtxLog.Infof("Individual PFD Management Transaction[%s] is removed", transID)
}

func (a *AfContext) IsAppIDExisted(appID string) (bool, string) {
	a.mtx.RLock()
	defer a.mtx.RUnlock()
	for _, pfdTrans := range a.pfdTrans {
		if pfdTrans.IsAppIDExisted(appID) {
			return true, pfdTrans.GetTransID()
		}
	}
	return false, ""
}

func (a *AfContext) GetAllSubsc() map[string]*AfSubscription {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	return a.subsc
}

func (a *AfContext) GetSubsc(subscID string) *AfSubscription {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	return a.subsc[subscID]
}

func (a *AfContext) DeleteSubsc(subscID string) {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	delete(a.subsc, subscID)
}
