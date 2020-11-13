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
	afs := AfSubscription{notifCorreID: strconv.FormatUint(numCorreID, 10)}
	a.mtx.Lock()
	a.numSubscID++
	afs.subscID = strconv.FormatUint(a.numSubscID, 10)
	a.mtx.Unlock()
	return &afs
}

func (a *AfContext) AddSubsc(afs *AfSubscription) {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	logger.CtxLog.Infof("New AF subscription[%s] added", afs.subscID)
	a.subsc[afs.subscID] = afs
}
