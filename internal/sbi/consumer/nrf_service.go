package consumer

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	nef_context "github.com/free5gc/nef/internal/context"
	"github.com/free5gc/nef/internal/logger"
	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/openapi/nrf/NFDisc"
	"github.com/free5gc/openapi/nrf/NFMgmt"
	sbi_metrics "github.com/free5gc/util/metrics/sbi"
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
}

func (s *nnrfService) getNFDiscoveryClient(uri string) *NFDisc.APIClient {
	s.nfDiscMu.RLock()
	if client, ok := s.nfDiscClients[uri]; ok {
		defer s.nfDiscMu.RUnlock()
		return client
	} else {
		configuration := NFDisc.NewConfiguration()
		configuration.SetBasePath(uri)
		configuration.SetMetrics(sbi_metrics.SbiMetricHook)
		configuration.SetHTTPClient(http.DefaultClient)
		cli := NFDisc.NewAPIClient(configuration)

		s.nfDiscMu.RUnlock()
		s.nfDiscMu.Lock()
		defer s.nfDiscMu.Unlock()
		s.nfDiscClients[uri] = cli
		return cli
	}
}

func (s *nnrfService) getNFManagementClient(uri string) *NFMgmt.APIClient {
	s.nfMngmntMu.RLock()
	if client, ok := s.nfMngmntClients[uri]; ok {
		defer s.nfMngmntMu.RUnlock()
		return client
	} else {
		configuration := NFMgmt.NewConfiguration()
		configuration.SetBasePath(uri)
		configuration.SetMetrics(sbi_metrics.SbiMetricHook)
		cli := NFMgmt.NewAPIClient(configuration)

		s.nfMngmntMu.RUnlock()
		s.nfMngmntMu.Lock()
		defer s.nfMngmntMu.Unlock()
		s.nfMngmntClients[uri] = cli
		return cli
	}
}

func (s *nnrfService) RegisterNFInstance(ctx context.Context, nefCtx *nef_context.NefContext) (
	resourceNrfUri string, retrieveNfInstanceId string, err error,
) {
	nfInstID := s.consumer.Context().NfInstID()
	nfProfile, err := s.buildNfProfile()
	if err != nil {
		return "", "", fmt.Errorf("failed to build NRF profile: %+v", err)
	}
	client := s.getNFManagementClient(s.consumer.Config().NrfUri())

	var nf models.Nrf_NFMgmt_NFProfile
	var res *NFMgmt.RegisterNFInstanceResponse
	finish := false
	for !finish {
		select {
		case <-ctx.Done():
			return "", "", fmt.Errorf("registration cancelled due to context cancellation")
		default:
			req := &NFMgmt.RegisterNFInstanceRequest{
				NfInstanceID: &nfInstID,
				RequestBody:  nfProfile,
			}

			res, err = client.NFInstanceIDDocumentApi.RegisterNFInstance(ctx, req)
			if err != nil || res == nil {
				logger.ConsumerLog.Infof("NEF register to NRF Error[%v]", err.Error())
				time.Sleep(RetryRegisterNrfDuration)
				continue
			}

			resourceUri := res.Location
			resourceNrfUri, _, _ = strings.Cut(resourceUri, "/nnrf-nfm/")
			retrieveNfInstanceId = resourceUri[strings.LastIndex(resourceUri, "/")+1:]
			if res.Nrf_NFMgmt_NFProfile != nil {
				nf = *res.Nrf_NFMgmt_NFProfile
			}

			oauth2 := false
			if customInfo, isMap := nf.CustomInfo.(map[string]interface{}); isMap {
				v, ok := customInfo["oauth2"].(bool)
				if ok {
					oauth2 = v
					logger.MainLog.Infoln("OAuth2 setting receive from NRF:", oauth2)
				}
			}
			if oauthErr := s.consumer.Context().SetOAuth2Required(oauth2); oauthErr != nil {
				return "", "", oauthErr
			}
			finish = true
		}
	}
	return resourceNrfUri, retrieveNfInstanceId, err
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
	profile.NfServices = make([]models.Nrf_NFMgmt_NFService, 0, len(nfServices))
	for _, nfService := range nfServices {
		allowed, known := nef_context.AllowedNfTypesForService(nfService.ServiceName)
		if !known {
			return nil, fmt.Errorf("no AllowedNfTypes policy for service %q", nfService.ServiceName)
		}
		nfService.AllowedNfTypes = allowed
		profile.NfServices = append(profile.NfServices, nfService)
	}
	return profile, nil
}

func (s *nnrfService) DeregisterNFInstance() (problemDetails *models.ProblemDetails, err error) {
	logger.ConsumerLog.Infof("DeregisterNFInstance")

	ctx, pd, err := s.consumer.Context().GetTokenCtxForNRF(models.Nrf_NFMgmt_ServiceName_NNRF_NFM)
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

	ctx, _, err := s.consumer.Context().GetTokenCtxForNRF(models.Nrf_NFMgmt_ServiceName_NNRF_DISC)
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
