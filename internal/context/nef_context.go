package context

import (
	"errors"
	"strconv"
	"sync"

	"github.com/google/uuid"

	"bitbucket.org/free5gc-team/nef/internal/logger"
	"bitbucket.org/free5gc-team/openapi/models"
)

type NefContext struct {
	nfInstID   string //NF Instance ID
	numCorreID uint64
	afCtxs     map[string]*AfContext
	mtx        sync.RWMutex
	pfdSubInfo PfdSubInfo
}

type PfdSubInfo struct {
	numPfdSubID   uint64
	appIdToSubIDs map[string]map[string]bool
	subIdToURI    map[string]string
}

func NewNefContext() *NefContext {
	n := &NefContext{nfInstID: uuid.New().String()}
	n.afCtxs = make(map[string]*AfContext)
	n.pfdSubInfo.appIdToSubIDs = make(map[string]map[string]bool)
	n.pfdSubInfo.subIdToURI = make(map[string]string)
	logger.CtxLog.Infof("New nfInstID: [%s]", n.nfInstID)
	return n
}

func (n *NefContext) GetNfInstID() string {
	n.mtx.RLock()
	defer n.mtx.RUnlock()
	return n.nfInstID
}

func (n *NefContext) SetNfInstID(id string) {
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

func (n *NefContext) AddAfCtx(afc *AfContext) {
	n.mtx.Lock()
	n.afCtxs[afc.afID] = afc
	n.mtx.Unlock()
	logger.CtxLog.Infof("New AF [%s] added", afc.afID)
}

func (n *NefContext) GetAfCtx(afID string) *AfContext {
	n.mtx.Lock()
	defer n.mtx.Unlock()
	return n.afCtxs[afID]
}

func (n *NefContext) DeleteAfCtx(afID string) {
	n.mtx.Lock()
	defer n.mtx.Unlock()
	if _, exist := n.afCtxs[afID]; !exist {
		logger.CtxLog.Infof("AF [%s] does not exist", afID)
		return
	}
	delete(n.afCtxs, afID)
}

func (n *NefContext) NewAfSubsc(afc *AfContext) *AfSubscription {
	n.mtx.Lock()
	defer n.mtx.Unlock()

	n.numCorreID++
	return afc.newSubsc(n.numCorreID)
}

func (n *NefContext) NewAfPfdTrans(afc *AfContext) *AfPfdTransaction {
	n.mtx.Lock()
	defer n.mtx.Unlock()
	return afc.newPfdTrans()
}

func (n *NefContext) IsAppIDExisted(appID string) (bool, string, string) {
	n.mtx.RLock()
	defer n.mtx.RUnlock()
	for _, afCtx := range n.afCtxs {
		if exist, transID := afCtx.IsAppIDExisted(appID); exist {
			return true, afCtx.GetAfID(), transID
		}
	}
	return false, "", ""
}

func (n *NefContext) GetAfCtxAndPfdTransWithTransID(afID, transID string) (*AfContext, *AfPfdTransaction, error) {
	afCtx := n.GetAfCtx(afID)
	if afCtx == nil {
		return nil, nil, errors.New("AF not found")
	}

	afPfdTrans := afCtx.GetPfdTrans(transID)
	if afPfdTrans == nil {
		return nil, nil, errors.New("Transaction not found")
	}

	return afCtx, afPfdTrans, nil
}

func (n *NefContext) GetPfdTransWithAppID(afID, transID, appID string) (*AfPfdTransaction, error) {
	_, afPfdTrans, err := n.GetAfCtxAndPfdTransWithTransID(afID, transID)
	if err != nil {
		return nil, err
	}

	if !afPfdTrans.IsAppIDExisted(appID) {
		return nil, errors.New("Application ID not found")
	}

	return afPfdTrans, nil
}

func (n *NefContext) AddPfdSub(pfdSub *models.PfdSubscription) string {
	n.mtx.Lock()
	defer n.mtx.Unlock()

	n.pfdSubInfo.numPfdSubID++
	subID := strconv.FormatUint(n.pfdSubInfo.numPfdSubID, 10)
	n.pfdSubInfo.subIdToURI[subID] = pfdSub.NotifyUri
	// TODO: If pfdSub.ApplicationIds is empty, it may means monitoring all appIDs
	for _, appID := range pfdSub.ApplicationIds {
		if _, exist := n.pfdSubInfo.appIdToSubIDs[appID]; !exist {
			n.pfdSubInfo.appIdToSubIDs[appID] = make(map[string]bool)
		}
		n.pfdSubInfo.appIdToSubIDs[appID][subID] = true
	}

	return subID
}

func (n *NefContext) DeletePfdSub(subID string) error {
	n.mtx.Lock()
	defer n.mtx.Unlock()

	if _, exist := n.pfdSubInfo.subIdToURI[subID]; !exist {
		return errors.New("Subscription not found")
	}
	delete(n.pfdSubInfo.subIdToURI, subID)
	for _, subIDs := range n.pfdSubInfo.appIdToSubIDs {
		delete(subIDs, subID)
	}
	return nil
}
