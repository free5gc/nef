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
	afPfdTrans := AfPfdTransaction{
		transID: strconv.FormatUint(a.numTransID, 10),
	}
	return &afPfdTrans
}

func (a *AfContext) AddPfdTrans(afPfdTrans *AfPfdTransaction) {
	a.mtx.Lock()
	a.pfdTrans[afPfdTrans.transID] = afPfdTrans
	a.mtx.Unlock()
	logger.CtxLog.Infof("New AF PFD transaction[%s] added", afPfdTrans.transID)
}
