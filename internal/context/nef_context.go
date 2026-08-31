package context

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/free5gc/nef/internal/logger"
	"github.com/free5gc/nef/pkg/factory"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/openapi/oauth"
	"github.com/google/uuid"
)

type nef interface {
	Config() *factory.Config
}

// NFContext is the interface used by middleware to perform inbound OAuth2 token checks.
type NFContext interface {
	AuthorizationCheck(token string, serviceName models.Nrf_NFMgmt_ServiceName) error
}

var _ NFContext = &NefContext{}

type NefContext struct {
	nef

	nfInstID        string // NF Instance ID
	nrfNfInstanceID string
	pcfPaUri        string
	udrDrUri        string
	numCorreID      uint64
	OAuth2Required  bool
	afs             map[string]*AfData
	mu              sync.RWMutex
}

func NewContext(nef nef) (*NefContext, error) {
	c := &NefContext{
		nef:      nef,
		nfInstID: nef.Config().GetNfInstanceId(),
	}
	c.afs = make(map[string]*AfData)
	logger.CtxLog.Infof("New nfInstID: [%s]", c.nfInstID)
	return c, nil
}

func (c *NefContext) NfInstID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.nfInstID
}

func (c *NefContext) SetNfInstID(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nfInstID = id
	logger.CtxLog.Infof("Set nfInstID: [%s]", c.nfInstID)
}

func (c *NefContext) PcfPaUri() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.pcfPaUri
}

func (c *NefContext) SetPcfPaUri(uri string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pcfPaUri = uri
	logger.CtxLog.Infof("Set pcfPaUri: [%s]", c.pcfPaUri)
}

func (c *NefContext) UdrDrUri() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.udrDrUri
}

func (c *NefContext) SetUdrDrUri(uri string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.udrDrUri = uri
	logger.CtxLog.Infof("Set udrDrUri: [%s]", c.udrDrUri)
}

func (c *NefContext) NewAf(afID string) *AfData {
	af := &AfData{
		AfID:     afID,
		Subs:     make(map[string]*AfSubscription),
		PfdTrans: make(map[string]*AfPfdTransaction),
		Log:      logger.CtxLog.WithField(logger.FieldAFID, fmt.Sprintf("AF:%s", afID)),
	}
	return af
}

func (c *NefContext) AddAf(af *AfData) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.afs[af.AfID] = af
	af.Log.Infoln("AF is added")
}

func (c *NefContext) GetAf(afID string) *AfData {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.afs[afID]
}

func (c *NefContext) DeleteAf(afID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.afs, afID)
	logger.CtxLog.Infof("AF[%s] is deleted", afID)
}

func (c *NefContext) NewCorreID() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.numCorreID++
	return c.numCorreID
}

func (c *NefContext) ResetCorreID() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.numCorreID = 0
}

func (c *NefContext) IsAppIDExisted(appID string) (string, string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, af := range c.afs {
		af.Mu.RLock()
		if transID, ok := af.IsAppIDExisted(appID); ok {
			defer af.Mu.RUnlock()
			return af.AfID, transID, true
		}
		af.Mu.RUnlock()
	}
	return "", "", false
}

func (c *NefContext) FindAfSub(CorrID string) (*AfData, *AfSubscription) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, af := range c.afs {
		af.Mu.RLock()
		for _, sub := range af.Subs {
			if sub.NotifCorreID == CorrID {
				defer af.Mu.RUnlock()
				return af, sub
			}
		}
		af.Mu.RUnlock()
	}
	return nil, nil
}

func (c *NefContext) GetTokenCtx(serviceName models.Nrf_NFMgmt_ServiceName, targetNF models.Nrf_NFMgmt_NFType) (
	context.Context, *models.ProblemDetails, error,
) {
	if !c.OAuth2Required {
		return context.TODO(), nil, nil
	}
	return oauth.GetTokenCtx(c.tokenRequest(serviceName, targetNF))
}

func (c *NefContext) GetTokenCtxForNFInstance(serviceName models.Nrf_NFMgmt_ServiceName,
	targetNF models.Nrf_NFMgmt_NFType, targetNFInstanceID string,
) (context.Context, *models.ProblemDetails, error) {
	if !c.OAuth2Required {
		return context.TODO(), nil, nil
	}
	targetID, err := uuid.Parse(strings.TrimSpace(targetNFInstanceID))
	if err != nil {
		return nil, nil, fmt.Errorf("invalid target NF instance ID: %w", err)
	}
	if targetID.Version() != 4 {
		return nil, nil, fmt.Errorf("invalid target NF instance ID: UUID must be version 4")
	}
	return oauth.GetTokenCtx(c.tokenRequestForNFInstance(serviceName, targetNF, targetNFInstanceID))
}

func (c *NefContext) GetTokenCtxForNRF(serviceName models.Nrf_NFMgmt_ServiceName) (
	context.Context, *models.ProblemDetails, error,
) {
	return c.GetTokenCtxForNFInstance(
		serviceName, models.Nrf_NFMgmt_NFType_NRF, c.nrfNfInstanceID)
}

func (c *NefContext) tokenRequest(serviceName models.Nrf_NFMgmt_ServiceName,
	targetNF models.Nrf_NFMgmt_NFType,
) oauth.TokenRequest {
	return oauth.TokenRequest{
		ConsumerNFType: models.Nrf_NFMgmt_NFType_NEF, ConsumerNFInstanceID: c.nfInstID,
		TargetNFType: targetNF, NRFURI: c.Config().NrfUri(), Scope: string(serviceName),
	}
}

func (c *NefContext) tokenRequestForNFInstance(serviceName models.Nrf_NFMgmt_ServiceName,
	targetNF models.Nrf_NFMgmt_NFType, targetNFInstanceID string,
) oauth.TokenRequest {
	request := c.tokenRequest(serviceName, targetNF)
	request.TargetNFInstanceID = targetNFInstanceID
	return request
}

func (c *NefContext) SetOAuth2Required(required bool) error {
	if !required {
		c.OAuth2Required = false
		c.nrfNfInstanceID = ""
		return nil
	}
	if strings.TrimSpace(c.Config().NrfCertPem()) == "" {
		return fmt.Errorf("OAuth2 enabled but NRF certificate path is empty")
	}
	if strings.TrimSpace(c.Config().NrfUri()) == "" {
		return fmt.Errorf("OAuth2 enabled but NRF URI is empty")
	}
	nrfNfInstanceID, err := oauth.NFInstanceIDFromCertificate(c.Config().NrfCertPem())
	if err != nil {
		return fmt.Errorf("derive trusted NRF instance ID from certificate: %w", err)
	}
	c.nrfNfInstanceID = nrfNfInstanceID
	c.OAuth2Required = true
	return nil
}

// AuthorizationCheck validates the inbound OAuth2 bearer token against serviceName.
// When OAuth2 is disabled it returns nil immediately (pass-through for dev/test).
func (c *NefContext) AuthorizationCheck(token string, serviceName models.Nrf_NFMgmt_ServiceName) error {
	if !c.OAuth2Required {
		logger.CtxLog.Debugf("NefContext::AuthorizationCheck: OAuth2 not required")
		return nil
	}
	logger.CtxLog.Debugf(
		"NefContext::AuthorizationCheck: tokenPresent[%t] tokenLen[%d] serviceName[%s]",
		token != "", len(token), serviceName,
	)
	return oauth.VerifyOAuth(token, string(serviceName), oauth.AudiencePolicy{
		NFInstanceID: c.nfInstID, NFType: models.Nrf_NFMgmt_NFType_NEF,
	}, c.nrfNfInstanceID, c.Config().NrfCertPem())
}
