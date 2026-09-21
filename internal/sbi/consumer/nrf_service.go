package consumer

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/free5gc/nef/internal/logger"
	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/openapi/nrf/NFDisc"
	"github.com/free5gc/openapi/nrf/NFMgmt"
	sbi_metrics "github.com/free5gc/util/metrics/sbi"
	"github.com/free5gc/util/nfheartbeat"
)

const (
	RetryRegisterNrfDuration = 2 * time.Second
)

var serviceNfType map[models.Nrf_NFMgmt_ServiceName]models.Nrf_NFMgmt_NFType

func init() {
	serviceNfType = make(map[models.Nrf_NFMgmt_ServiceName]models.Nrf_NFMgmt_NFType)
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NNRF_NFM] = models.Nrf_NFMgmt_NFType_NRF
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NNRF_DISC] = models.Nrf_NFMgmt_NFType_NRF
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NUDM_SDM] = models.Nrf_NFMgmt_NFType_UDM
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NUDM_UECM] = models.Nrf_NFMgmt_NFType_UDM
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NUDM_UEAU] = models.Nrf_NFMgmt_NFType_UDM
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NUDM_EE] = models.Nrf_NFMgmt_NFType_UDM
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NUDM_PP] = models.Nrf_NFMgmt_NFType_UDM
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NAMF_COMM] = models.Nrf_NFMgmt_NFType_AMF
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NAMF_EVTS] = models.Nrf_NFMgmt_NFType_AMF
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NAMF_MT] = models.Nrf_NFMgmt_NFType_AMF
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NAMF_LOC] = models.Nrf_NFMgmt_NFType_AMF
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NSMF_PDUSESSION] = models.Nrf_NFMgmt_NFType_SMF
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NSMF_EVENT_EXPOSURE] = models.Nrf_NFMgmt_NFType_SMF
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NAUSF_AUTH] = models.Nrf_NFMgmt_NFType_AUSF
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NAUSF_SORPROTECTION] = models.Nrf_NFMgmt_NFType_AUSF
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NAUSF_UPUPROTECTION] = models.Nrf_NFMgmt_NFType_AUSF
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NNEF_PFDMANAGEMENT] = models.Nrf_NFMgmt_NFType_NEF
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NPCF_AM_POLICY_CONTROL] = models.Nrf_NFMgmt_NFType_PCF
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NPCF_SMPOLICYCONTROL] = models.Nrf_NFMgmt_NFType_PCF
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NPCF_POLICYAUTHORIZATION] = models.Nrf_NFMgmt_NFType_PCF
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NPCF_BDTPOLICYCONTROL] = models.Nrf_NFMgmt_NFType_PCF
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NPCF_EVENTEXPOSURE] = models.Nrf_NFMgmt_NFType_PCF
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NPCF_UE_POLICY_CONTROL] = models.Nrf_NFMgmt_NFType_PCF
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NSMSF_SMS] = models.Nrf_NFMgmt_NFType_SMSF
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NNSSF_NSSELECTION] = models.Nrf_NFMgmt_NFType_NSSF
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NNSSF_NSSAIAVAILABILITY] = models.Nrf_NFMgmt_NFType_NSSF
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NUDR_DR] = models.Nrf_NFMgmt_NFType_UDR
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NLMF_LOC] = models.Nrf_NFMgmt_NFType_LMF
	serviceNfType[models.Nrf_NFMgmt_ServiceName_N5G_EIR_EIC] = models.Nrf_NFMgmt_NFType_5_G_EIR
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NBSF_MANAGEMENT] = models.Nrf_NFMgmt_NFType_BSF
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NCHF_SPENDINGLIMITCONTROL] = models.Nrf_NFMgmt_NFType_CHF
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NCHF_CONVERGEDCHARGING] = models.Nrf_NFMgmt_NFType_CHF
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NNWDAF_EVENTSSUBSCRIPTION] = models.Nrf_NFMgmt_NFType_NWDAF
	serviceNfType[models.Nrf_NFMgmt_ServiceName_NNWDAF_ANALYTICSINFO] = models.Nrf_NFMgmt_NFType_NWDAF
}

type nnrfService struct {
	consumer *Consumer

	nfDiscMu      sync.RWMutex
	nfDiscClients map[string]*NFDisc.APIClient

	nfMngmntMu      sync.RWMutex
	nfMngmntClients map[string]*NFMgmt.APIClient

	heartbeat *nfheartbeat.Runner

	// Interval in seconds from the last registration response. Written before the heartbeat starts, then only on
	// the heartbeat goroutine.
	heartbeatTimer int32
}

func (s *nnrfService) getNFDiscoveryClient(uri string) *NFDisc.APIClient {
	s.nfDiscMu.RLock()
	client, ok := s.nfDiscClients[uri]
	s.nfDiscMu.RUnlock()
	if ok {
		return client
	}

	s.nfDiscMu.Lock()
	defer s.nfDiscMu.Unlock()
	// Another caller may have stored a client while the read lock was down.
	if client, ok = s.nfDiscClients[uri]; ok {
		return client
	}

	configuration := NFDisc.NewConfiguration()
	configuration.SetBasePath(uri)
	configuration.SetMetrics(sbi_metrics.SbiMetricHook)
	configuration.SetHTTPClient(http.DefaultClient)
	client = NFDisc.NewAPIClient(configuration)

	s.nfDiscClients[uri] = client
	return client
}

func (s *nnrfService) getNFManagementClient(uri string) *NFMgmt.APIClient {
	s.nfMngmntMu.RLock()
	client, ok := s.nfMngmntClients[uri]
	s.nfMngmntMu.RUnlock()
	if ok {
		return client
	}

	s.nfMngmntMu.Lock()
	defer s.nfMngmntMu.Unlock()
	// Another caller may have stored a client while the read lock was down.
	if client, ok = s.nfMngmntClients[uri]; ok {
		return client
	}

	configuration := NFMgmt.NewConfiguration()
	configuration.SetBasePath(uri)
	configuration.SetMetrics(sbi_metrics.SbiMetricHook)
	client = NFMgmt.NewAPIClient(configuration)

	s.nfMngmntClients[uri] = client
	return client
}

// RegisterNFInstance PUTs the profile on the NF's own instance ID (3GPP TS 29.510 clause 5.2.2.2.2) until it
// succeeds or ctx is done. Only the startup call may pass applyOAuth2: it writes OAuth2Required unsynchronized.
func (s *nnrfService) RegisterNFInstance(ctx context.Context, applyOAuth2 bool) error {
	nfInstID := s.consumer.Context().NfInstID()
	nfProfile, err := s.buildNfProfile()
	if err != nil {
		return fmt.Errorf("failed to build NRF profile: %+v", err)
	}
	client := s.getNFManagementClient(s.consumer.Config().NrfUri())

	var res *NFMgmt.RegisterNFInstanceResponse
	req := &NFMgmt.RegisterNFInstanceRequest{
		NfInstanceID: &nfInstID,
		RequestBody:  nfProfile,
	}
	for ctx.Err() == nil {
		res, err = client.NFInstanceIDDocumentApi.RegisterNFInstance(ctx, req)
		if err == nil && res != nil {
			s.processRegisterResponse(res.Nrf_NFMgmt_NFProfile, applyOAuth2)
			return nil
		}
		logger.ConsumerLog.Errorf("NEF register to NRF Error[%v]", err)
		select {
		case <-ctx.Done():
		case <-time.After(RetryRegisterNrfDuration):
		}
	}
	return fmt.Errorf("NFRegister aborted: %w (last error: %v)", ctx.Err(), err)
}

// processRegisterResponse adopts what the NRF answered to the NFRegister PUT: the
// heartbeat interval and the oauth2 custom info.
func (s *nnrfService) processRegisterResponse(nf *models.Nrf_NFMgmt_NFProfile, applyOAuth2 bool) {
	if nf == nil {
		nf = &models.Nrf_NFMgmt_NFProfile{}
	}
	s.heartbeatTimer = nf.HeartBeatTimer

	oauth2 := false
	if customInfo, isMap := nf.CustomInfo.(map[string]interface{}); isMap {
		if v, ok := customInfo["oauth2"].(bool); ok {
			oauth2 = v
			logger.MainLog.Infoln("OAuth2 setting receive from NRF:", oauth2)
		}
	}
	nefContext := s.consumer.Context()
	if applyOAuth2 {
		nefContext.OAuth2Required = oauth2
		if oauth2 && s.consumer.Config().NrfCertPem() == "" {
			logger.CfgLog.Error("OAuth2 enable but no nrfCertPem provided in config.")
		}
	} else if oauth2 != nefContext.OAuth2Required {
		logger.ConsumerLog.Warnf("NRF OAuth2 setting changed to %v, restart NEF to apply it", oauth2)
	}
}

// UpdateNFInstance sends an NFUpdate PATCH to the NRF, honoring ctx. The raw err
// comes back alongside any ProblemDetails so callers can read its
// GenericOpenAPIError status.
func (s *nnrfService) UpdateNFInstance(ctx context.Context, patchItems []models.PatchItem) (
	nf models.Nrf_NFMgmt_NFProfile, problemDetails *models.ProblemDetails, err error,
) {
	tokCtx, pd, err := s.consumer.Context().GetTokenCtx(
		models.Nrf_NFMgmt_ServiceName_NNRF_NFM,
		models.Nrf_NFMgmt_NFType_NRF,
	)
	if err != nil {
		return nf, pd, err
	}
	// GetTokenCtx takes no parent, so the token request stays uncancelable;
	// transplanting the token lets at least the PATCH honor ctx.
	if tok := tokCtx.Value(openapi.ContextOAuth2); tok != nil {
		ctx = context.WithValue(ctx, openapi.ContextOAuth2, tok)
	}

	client := s.getNFManagementClient(s.consumer.Config().NrfUri())
	nfInstanceId := s.consumer.Context().NfInstID()
	request := &NFMgmt.UpdateNFInstanceRequest{
		NfInstanceID: &nfInstanceId,
		RequestBody:  patchItems,
	}

	res, err := client.NFInstanceIDDocumentApi.UpdateNFInstance(ctx, request)
	if err != nil {
		var apiErr openapi.GenericOpenAPIError
		if errors.As(err, &apiErr) {
			if updateErr, okModel := apiErr.Model().(NFMgmt.UpdateNFInstanceError); okModel {
				return nf, updateErr.ProblemDetails, err
			}
		}
		return nf, nil, err
	}
	if res == nil {
		return nf, nil, fmt.Errorf("empty NFUpdate response")
	}
	if res.Nrf_NFMgmt_NFProfile != nil {
		nf = *res.Nrf_NFMgmt_NFProfile
	}
	return nf, nil, nil
}

func (s *nnrfService) buildNfProfile() (
	profile *models.Nrf_NFMgmt_NFProfile, err error,
) {
	profile = &models.Nrf_NFMgmt_NFProfile{}

	profile.NfInstanceId = s.consumer.Context().NfInstID()
	profile.NfType = models.Nrf_NFMgmt_NFType_NEF
	profile.NfStatus = models.Nrf_NFMgmt_NFStatus_REGISTERED

	cfg := s.consumer.Config()
	profile.Ipv4Addresses = append(profile.Ipv4Addresses, cfg.SbiRegisterIP())
	nfServices := cfg.NFServices()
	if len(nfServices) == 0 {
		return nil, fmt.Errorf("buildNfProfile err: NFServices is Empty")
	}
	profile.NfServices = nfServices
	return profile, nil
}

func (s *nnrfService) DeregisterNFInstance() (problemDetails *models.ProblemDetails, err error) {
	logger.ConsumerLog.Infof("DeregisterNFInstance")

	ctx, pd, err := s.consumer.Context().GetTokenCtx(models.Nrf_NFMgmt_ServiceName_NNRF_NFM, models.Nrf_NFMgmt_NFType_NRF)
	if err != nil {
		return pd, err
	}

	client := s.getNFManagementClient(s.consumer.Config().NrfUri())

	nfInstanceId := s.consumer.Context().NfInstID()
	req := &NFMgmt.DeregisterNFInstanceRequest{
		NfInstanceID: &nfInstanceId,
	}

	_, err = client.NFInstanceIDDocumentApi.DeregisterNFInstance(ctx, req)
	if err != nil {
		switch apiErr := err.(type) {
		// API error
		case openapi.GenericOpenAPIError:
			switch errModel := apiErr.Model().(type) {
			case NFMgmt.DeregisterNFInstanceError:
				problemDetails = errModel.ProblemDetails
			case error:
				problemDetails = openapi.ProblemDetailsSystemFailure(errModel.Error())
			default:
				err = openapi.ReportError("openapi error")
			}
		case error:
			problemDetails = openapi.ProblemDetailsSystemFailure(apiErr.Error())
		default:
			err = openapi.ReportError("server no response")
		}
	}
	return problemDetails, err
}

func (s *nnrfService) SearchNFInstances(nrfUri string, srvName models.Nrf_NFMgmt_ServiceName, targetNfType,
	requestNfType models.Nrf_NFMgmt_NFType, param *NFDisc.SearchNFInstancesRequest,
) (*models.Nrf_NFDisc_NFProfile, string, error) {
	client := s.getNFDiscoveryClient(nrfUri)

	if client == nil {
		return nil, "", openapi.ReportError("nrf not found")
	}

	ctx, _, err := s.consumer.Context().GetTokenCtx(models.Nrf_NFMgmt_ServiceName_NNRF_DISC, models.Nrf_NFMgmt_NFType_NRF)
	if err != nil {
		return nil, "", err
	}

	param.TargetNfType = &targetNfType
	param.RequesterNfType = &requestNfType
	res, err := client.NFInstancesStoreApi.SearchNFInstances(ctx, param)
	if err != nil {
		logger.ConsumerLog.Errorf("SearchNFInstances failed: %+v", err)
		return nil, "", err
	}
	// The search result is a pointer in the new openapi models, so it can be
	// nil even when the call itself succeeded.
	if res == nil || res.Nrf_NFDisc_SearchResult == nil {
		return nil, "", openapi.ReportError("no search result from NRF")
	}

	nfProf, uri, err := getProfileAndUri(res.Nrf_NFDisc_SearchResult, srvName)
	if err != nil {
		logger.ConsumerLog.Errorf("%s", err.Error())
		return nil, "", err
	}
	return nfProf, uri, nil
}

func getProfileAndUri(resp *models.Nrf_NFDisc_SearchResult, srvName models.Nrf_NFMgmt_ServiceName) (
	*models.Nrf_NFDisc_NFProfile, string, error,
) {
	// select the first ServiceName
	// TODO: select base on other info
	var profile *models.Nrf_NFDisc_NFProfile
	var uri string
	for _, nfProfile := range resp.NfInstances {
		profile = &nfProfile
		uri = searchNFServiceUri(nfProfile, srvName, models.Nrf_NFMgmt_NFServiceStatus_REGISTERED)
		if uri != "" {
			break
		}
	}
	if uri == "" {
		return nil, "", fmt.Errorf("no uri for %s found", srvName)
	}
	return profile, uri, nil
}

// searchNFServiceUri returns NF Uri derived from NfProfile with corresponding service
func searchNFServiceUri(nfProfile models.Nrf_NFDisc_NFProfile, serviceName models.Nrf_NFMgmt_ServiceName,
	nfServiceStatus models.Nrf_NFMgmt_NFServiceStatus,
) string {
	if nfProfile.NfServices == nil {
		return ""
	}

	nfUri := ""
	for _, service := range nfProfile.NfServices {
		if service.ServiceName == serviceName && service.NfServiceStatus == nfServiceStatus {
			if service.Fqdn != "" {
				nfUri = string(service.Scheme) + "://" + service.Fqdn
			} else if nfProfile.Fqdn != "" {
				nfUri = string(service.Scheme) + "://" + nfProfile.Fqdn
			} else if service.ApiPrefix != "" {
				u, err := url.Parse(service.ApiPrefix)
				if err != nil {
					return nfUri
				}
				nfUri = u.Scheme + "://" + u.Host
			} else if service.IpEndPoints != nil {
				point := (service.IpEndPoints)[0]
				if point.Ipv4Address != "" {
					nfUri = getUriFromIpEndPoint(service.Scheme, point.Ipv4Address, point.Port)
				} else if len(nfProfile.Ipv4Addresses) != 0 {
					nfUri = getUriFromIpEndPoint(service.Scheme, nfProfile.Ipv4Addresses[0], point.Port)
				}
			}
		}
		if nfUri != "" {
			break
		}
	}

	return nfUri
}

func getUriFromIpEndPoint(scheme models.UriScheme, ipv4Address string, port int32) string {
	uri := ""
	if port != 0 {
		uri = string(scheme) + "://" + ipv4Address + ":" + strconv.Itoa(int(port))
	} else {
		switch scheme {
		case models.UriScheme_HTTP:
			uri = string(scheme) + "://" + ipv4Address + ":80"
		case models.UriScheme_HTTPS:
			uri = string(scheme) + "://" + ipv4Address + ":443"
		}
	}
	return uri
}
