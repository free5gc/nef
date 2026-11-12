package notifier

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"github.com/free5gc/nef/internal/logger"
	// "github.com/free5gc/openapi/Nnef_PFDmanagement"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/openapi/nef/PFDMgmt"
	sbi_metrics "github.com/free5gc/util/metrics/sbi"
)

type PfdChangeNotifier struct {
	verify              bool
	clientPfdManagement *PFDMgmt.APIClient
	mu                  sync.RWMutex

	numPfdSubID   uint64
	appIdToSubIDs map[string]map[string]bool
	subIdToURI    map[string]string
}

type PfdNotifyContext struct {
	notifier             *PfdChangeNotifier
	appIdToNotification  map[string]models.Nef_PFDMgmt_PfdChangeNotification
	subIdToChangedAppIDs map[string][]string
}

func NewPfdChangeNotifier(verify bool) (*PfdChangeNotifier, error) {
	return &PfdChangeNotifier{
		verify:        verify,
		appIdToSubIDs: make(map[string]map[string]bool),
		subIdToURI:    make(map[string]string),
	}, nil
}

func (n *PfdChangeNotifier) initPfdManagementApiClient() {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.clientPfdManagement != nil {
		return
	}

	config := PFDMgmt.NewConfiguration()
	config.SetMetrics(sbi_metrics.SbiMetricHook)
	config.SetHTTPClient(&http.Client{Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: !n.verify}, // #nosec G402 -- NF configuration
	}})
	n.clientPfdManagement = PFDMgmt.NewAPIClient(config)
}

func (n *PfdChangeNotifier) AddPfdSub(pfdSub *models.Nef_PFDMgmt_PfdSubscription) string {
	n.initPfdManagementApiClient()

	n.mu.Lock()
	defer n.mu.Unlock()

	n.numPfdSubID++
	subID := strconv.FormatUint(n.numPfdSubID, 10)
	n.subIdToURI[subID] = pfdSub.NotifyUri
	// TODO: If pfdSub.ApplicationIds is empty, it may means monitoring all appIDs
	for _, appID := range pfdSub.ApplicationIds {
		if _, exist := n.appIdToSubIDs[appID]; !exist {
			n.appIdToSubIDs[appID] = make(map[string]bool)
		}
		n.appIdToSubIDs[appID][subID] = true
	}

	return subID
}

func (n *PfdChangeNotifier) DeletePfdSub(subID string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if _, exist := n.subIdToURI[subID]; !exist {
		return errors.New("subscription not found")
	}
	delete(n.subIdToURI, subID)
	for _, subIDs := range n.appIdToSubIDs {
		delete(subIDs, subID)
	}
	return nil
}

func (n *PfdChangeNotifier) getSubIDs(appID string) []string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	subIDs := make([]string, 0, len(n.appIdToSubIDs[appID]))
	for subID := range n.appIdToSubIDs[appID] {
		subIDs = append(subIDs, subID)
	}
	return subIDs
}

func (n *PfdChangeNotifier) getSubURI(subID string) string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.subIdToURI[subID]
}

func (n *PfdChangeNotifier) NewPfdNotifyContext() *PfdNotifyContext {
	return &PfdNotifyContext{
		notifier:             n,
		appIdToNotification:  make(map[string]models.Nef_PFDMgmt_PfdChangeNotification),
		subIdToChangedAppIDs: make(map[string][]string),
	}
}

func (nc *PfdNotifyContext) AddNotification(appID string, notif *models.Nef_PFDMgmt_PfdChangeNotification) {
	nc.appIdToNotification[appID] = *notif
	for _, subID := range nc.notifier.getSubIDs(appID) {
		nc.subIdToChangedAppIDs[subID] = append(nc.subIdToChangedAppIDs[subID], appID)
	}
}

func (nc *PfdNotifyContext) FlushNotifications() {
	for subID, appIDs := range nc.subIdToChangedAppIDs {
		pfdChangeNotifications := make([]models.Nef_PFDMgmt_PfdChangeNotification, 0, len(appIDs))
		for _, appID := range appIDs {
			pfdChangeNotifications = append(pfdChangeNotifications, nc.appIdToNotification[appID])
		}

		notifyReq := &PFDMgmt.NnefPFDmanagementNotifyRequest{
			RequestBody: pfdChangeNotifications,
		}

		go nc.notifier.sendPFDChangeNotification(subID, notifyReq)
		// TODO: Handle the response of notification properly
	}
}

func (n *PfdChangeNotifier) sendPFDChangeNotification(
	subID string,
	notifyReq *PFDMgmt.NnefPFDmanagementNotifyRequest,
) {
	defer func() {
		if p := recover(); p != nil {
			// Async callback failures must stay local and must not terminate NEF.
			logger.PFDManageLog.Errorf("panic while sending PFD notification: %v\n%s", p, string(debug.Stack()))
		}
	}()

	uri := n.getSubURI(subID)
	_, err := n.clientPfdManagement.PFDSubscriptionsApi.NnefPFDmanagementNotify(context.TODO(), uri, notifyReq)
	if err != nil {
		logger.PFDManageLog.Errorf("PFD notification delivery failed, subID[%s] notifyUri[%s], err[%+v]", subID, uri, err)
	}
}
