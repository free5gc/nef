package context

import (
	"sync"

	"github.com/google/uuid"

	"bitbucket.org/free5gc-team/nef/internal/logger"
)

type NefContext struct {
	nfInstID   string //NF Instance ID
	numCorreID uint64
	afCtxs     map[string]*AfContext
	mtx        sync.RWMutex
}

func NewNefContext() *NefContext {
	n := &NefContext{nfInstID: uuid.New().String()}
	n.afCtxs = make(map[string]*AfContext)
	logger.CtxLog.Infof("New nfInstID: [%s]", n.nfInstID)
	return n
}

func (n *NefContext) GetNfInstID() string {
	n.mtx.RLock()
	defer n.mtx.RUnlock()
	return n.nfInstID
}

func (n *NefContext) NfInstID(id string) {
	n.mtx.Lock()
	defer n.mtx.Unlock()
	n.nfInstID = id
	logger.CtxLog.Infof("Set nfInstID: [%s]", n.nfInstID)
}

func (n *NefContext) NewAfCtx(afID string) *AfContext {
	n.mtx.RLock()
	afc, exist := n.afCtxs[afID]
	n.mtx.RUnlock()
	if exist {
		logger.CtxLog.Infof("AF [%s] found", afID)
		return afc
	}

	logger.CtxLog.Infof("No AF found - new AF [%s]", afID)
	afc = &AfContext{afID: afID}
	afc.subsc = make(map[string]*AfSubscription)
	afc.pfdTrans = make(map[string]*AfPfdTransaction)
	return afc
}

func (n *NefContext) NewAfSubsc(afc *AfContext) *AfSubscription {
	n.mtx.Lock()
	defer n.mtx.Unlock()
	n.numCorreID++
	return afc.newSubsc(n.numCorreID)
}

func (n *NefContext) AddAfCtx(afc *AfContext) {
	n.mtx.Lock()
	defer n.mtx.Unlock()

	logger.CtxLog.Infof("New AF [%s] added", afc.afID)
	n.afCtxs[afc.afID] = afc
}
