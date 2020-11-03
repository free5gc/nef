package context

import (
	"sync"

	"github.com/google/uuid"

	"bitbucket.org/free5gc-team/nef/internal/logger"
)

type NefContext struct {
	nfInstID string //NF Instance ID
	pcfURI   string //PCF URI discovered from NRF
	udrURI   string //UDR URI discovered from NRF
	afCtx    map[string]*afContext
	mtx      sync.RWMutex
}

type afContext struct {
	subsc    map[string]*subscription
	pfdTrans map[string]*pfdTransaction
}

type subscription struct {
}

type pfdTransaction struct {
}

func NewNefContext() *NefContext {
	n := &NefContext{nfInstID: uuid.New().String()}
	n.afCtx = make(map[string]*afContext)
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
	n.nfInstID = id
	logger.CtxLog.Infof("Set nfInstID: [%s]", n.nfInstID)
	n.mtx.Unlock()
}

func (n *NefContext) GetPcfURI() string {
	n.mtx.RLock()
	defer n.mtx.RUnlock()
	return n.pcfURI
}

func (n *NefContext) PcfURI(uri string) {
	n.mtx.Lock()
	n.pcfURI = uri
	logger.CtxLog.Infof("Set pcfURI: [%s]", n.pcfURI)
	n.mtx.Unlock()
}

func (n *NefContext) GetUdrURI() string {
	n.mtx.RLock()
	defer n.mtx.RUnlock()
	return n.udrURI
}

func (n *NefContext) UdrURI(uri string) {
	n.mtx.Lock()
	n.udrURI = uri
	logger.CtxLog.Infof("Set udrURI: [%s]", n.udrURI)
	n.mtx.Unlock()
}
