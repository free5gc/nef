package context

import (
	"errors"
	"sync"

	"github.com/google/uuid"

	"bitbucket.org/free5gc-team/nef/internal/logger"
	"bitbucket.org/free5gc-team/nef/pkg/factory"
)

type nef interface {
	Config() *factory.Config
}

type NefContext struct {
	nef

	nfInstID   string // NF Instance ID
	pcfPaUri   string
	udrDrUri   string
	numCorreID uint64
	afCtxs     map[string]*AfContext
	mtx        sync.RWMutex
}

func NewNefContext(nef nef) (*NefContext, error) {
	c := &NefContext{
		nef:      nef,
		nfInstID: uuid.New().String(),
	}
	c.afCtxs = make(map[string]*AfContext)
	logger.CtxLog.Infof("New nfInstID: [%s]", c.nfInstID)
	return c, nil
}

func (c *NefContext) NfInstID() string {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.nfInstID
}

func (c *NefContext) SetNfInstID(id string) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.nfInstID = id
	logger.CtxLog.Infof("Set nfInstID: [%s]", c.nfInstID)
}

func (c *NefContext) PcfPaUri() string {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.pcfPaUri
}

func (c *NefContext) SetPcfPaUri(uri string) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.pcfPaUri = uri
	logger.CtxLog.Infof("Set pcfPaUri: [%s]", c.pcfPaUri)
}

func (c *NefContext) UdrDrUri() string {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.udrDrUri
}

func (c *NefContext) SetUdrDrUri(uri string) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.udrDrUri = uri
	logger.CtxLog.Infof("Set udrDrUri: [%s]", c.udrDrUri)
}

func (c *NefContext) NewAfCtx(afID string) *AfContext {
	c.mtx.RLock()
	afc, exist := c.afCtxs[afID]
	c.mtx.RUnlock()
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

func (c *NefContext) AddAfCtx(afc *AfContext) {
	c.mtx.Lock()
	c.afCtxs[afc.afID] = afc
	c.mtx.Unlock()
	logger.CtxLog.Infof("New AF [%s] added", afc.afID)
}

func (c *NefContext) GetAfCtx(afID string) *AfContext {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	return c.afCtxs[afID]
}

func (c *NefContext) DeleteAfCtx(afID string) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if _, exist := c.afCtxs[afID]; !exist {
		logger.CtxLog.Infof("AF [%s] does not exist", afID)
		return
	}
	delete(c.afCtxs, afID)
}

func (c *NefContext) NewAfSubsc(afc *AfContext) *AfSubscription {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	c.numCorreID++
	return afc.newSubsc(c.numCorreID)
}

func (c *NefContext) NewAfPfdTrans(afc *AfContext) *AfPfdTransaction {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	return afc.newPfdTrans()
}

func (c *NefContext) IsAppIDExisted(appID string) (bool, string, string) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	for _, afCtx := range c.afCtxs {
		if exist, transID := afCtx.IsAppIDExisted(appID); exist {
			return true, afCtx.GetAfID(), transID
		}
	}
	return false, "", ""
}

func (c *NefContext) GetAfCtxAndPfdTransWithTransID(afID, transID string) (*AfContext, *AfPfdTransaction, error) {
	afCtx := c.GetAfCtx(afID)
	if afCtx == nil {
		return nil, nil, errors.New("AF not found")
	}

	afPfdTrans := afCtx.GetPfdTrans(transID)
	if afPfdTrans == nil {
		return nil, nil, errors.New("Transaction not found")
	}

	return afCtx, afPfdTrans, nil
}

func (c *NefContext) GetPfdTransWithAppID(afID, transID, appID string) (*AfPfdTransaction, error) {
	_, afPfdTrans, err := c.GetAfCtxAndPfdTransWithTransID(afID, transID)
	if err != nil {
		return nil, err
	}

	if !afPfdTrans.IsAppIDExisted(appID) {
		return nil, errors.New("Application ID not found")
	}

	return afPfdTrans, nil
}
