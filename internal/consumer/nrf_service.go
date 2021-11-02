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

	"bitbucket.org/free5gc-team/nef/internal/logger"
	"bitbucket.org/free5gc-team/openapi/Nnrf_NFDiscovery"
	"bitbucket.org/free5gc-team/openapi/Nnrf_NFManagement"
	"bitbucket.org/free5gc-team/openapi/models"
	"github.com/antihax/optional"
)

const (
	RetryRegisterNrfDuration = 2 * time.Second
)

type nnrfService struct {
	consumer *Consumer

	nfDiscMu      sync.RWMutex
	nfDiscClients map[string]*Nnrf_NFDiscovery.APIClient

	nfMngmntMu      sync.RWMutex
	nfMngmntClients map[string]*Nnrf_NFManagement.APIClient
}

func (s *nnrfService) getNFDiscoveryClient(uri string) *Nnrf_NFDiscovery.APIClient {
	s.nfDiscMu.RLock()
	if client, ok := s.nfDiscClients[uri]; ok {
		defer s.nfDiscMu.RUnlock()
		return client
	} else {
		configuration := Nnrf_NFDiscovery.NewConfiguration()
		configuration.SetBasePath(uri)
		cli := Nnrf_NFDiscovery.NewAPIClient(configuration)

		s.nfDiscMu.RUnlock()
		s.nfDiscMu.Lock()
		defer s.nfDiscMu.Unlock()
		s.nfDiscClients[uri] = cli
		return cli
	}
}

func (s *nnrfService) getNFManagementClient(uri string) *Nnrf_NFManagement.APIClient {
	s.nfMngmntMu.RLock()
	if client, ok := s.nfMngmntClients[uri]; ok {
		defer s.nfMngmntMu.RUnlock()
		return client
	} else {
		configuration := Nnrf_NFManagement.NewConfiguration()
		configuration.SetBasePath(uri)
		cli := Nnrf_NFManagement.NewAPIClient(configuration)

		s.nfMngmntMu.RUnlock()
		s.nfMngmntMu.Lock()
		defer s.nfMngmntMu.Unlock()
		s.nfMngmntClients[uri] = cli
		return cli
	}
}

func (s *nnrfService) RegisterNFInstance() error {
	var rsp *http.Response
	var err error

	client := s.getNFManagementClient(s.consumer.Config().NrfUri())
	nfProfile, err := s.buildNfProfile()
	if err != nil {
		return fmt.Errorf("RegisterNFInstance err: %+v", err)
	}

	for {
		_, rsp, err = client.NFInstanceIDDocumentApi.RegisterNFInstance(
			context.TODO(), s.consumer.Context().NfInstID(), *nfProfile)
		if rsp != nil && rsp.Body != nil {
			if bodyCloseErr := rsp.Body.Close(); bodyCloseErr != nil {
				logger.ConsumerLog.Errorf("response body cannot close: %+v", bodyCloseErr)
			}
		}

		if err != nil || rsp == nil {
			logger.ConsumerLog.Infof("AMF register to NRF Error[%v], sleep 2s and retry", err)
			time.Sleep(RetryRegisterNrfDuration)
			continue
		}

		status := rsp.StatusCode
		if status == http.StatusOK {
			// NFUpdate
			logger.ConsumerLog.Infof("NFRegister Update")
			break
		} else if status == http.StatusCreated {
			// NFRegister
			resourceUri := rsp.Header.Get("Location")
			// resouceNrfUri := resourceUri[:strings.Index(resourceUri, "/nnrf-nfm/")]
			s.consumer.Context().SetNfInstID(resourceUri[strings.LastIndex(resourceUri, "/")+1:])
			logger.ConsumerLog.Infof("NFRegister Created")
			break
		} else {
			logger.ConsumerLog.Infof("NRF return wrong status: %d", status)
		}
	}
	return nil
}

func (s *nnrfService) buildNfProfile() (*models.NfProfile, error) {
	profile := &models.NfProfile{
		NfInstanceId: s.consumer.Context().NfInstID(),
		NfType:       models.NfType_NEF,
		NfStatus:     models.NfStatus_REGISTERED,
	}

	cfg := s.consumer.Config()
	profile.Ipv4Addresses = append(profile.Ipv4Addresses, cfg.SbiRegisterIP())

	versions := strings.Split(cfg.Version(), ".")
	majorVersionUri := "v" + versions[0]
	nfServices := []models.NfService{}
	for i, service := range cfg.ServiceList() {
		nfService := models.NfService{
			ServiceInstanceId: strconv.Itoa(i),
			ServiceName:       models.ServiceName(service.ServiceName),
			Versions: &[]models.NfServiceVersion{
				{
					ApiFullVersion:  cfg.Version(),
					ApiVersionInUri: majorVersionUri,
				},
			},
			Scheme:          models.UriScheme(cfg.SbiScheme()),
			NfServiceStatus: models.NfServiceStatus_REGISTERED,
			ApiPrefix:       cfg.SbiUri(),
			IpEndPoints: &[]models.IpEndPoint{
				{
					Ipv4Address: cfg.SbiRegisterIP(),
					Transport:   models.TransportProtocol_TCP,
					Port:        int32(cfg.SbiPort()),
				},
			},
			SupportedFeatures: service.SuppFeat,
		}
		nfServices = append(nfServices, nfService)
	}
	profile.NfServices = &nfServices
	return profile, nil
}

func (s *nnrfService) SearchNFInstances(nrfUri string, targetNfType models.NfType,
	param *Nnrf_NFDiscovery.SearchNFInstancesParamOpts) (models.SearchResult, error) {
	client := s.getNFDiscoveryClient(nrfUri)

	result, rsp, err := client.NFInstancesStoreApi.SearchNFInstances(context.TODO(),
		targetNfType, models.NfType_NEF, param)
	if rsp != nil && rsp.Body != nil {
		if bodyCloseErr := rsp.Body.Close(); bodyCloseErr != nil {
			logger.ConsumerLog.Errorf("SearchNFInstances err: response body cannot close: %+v", bodyCloseErr)
		}
	}
	if rsp != nil && rsp.StatusCode == http.StatusTemporaryRedirect {
		err = fmt.Errorf("SearchNFInstances err: Temporary Redirect")
	}
	return result, err
}

func (s *nnrfService) SearchPcfPolicyAuthUri() (string, error) {
	param := Nnrf_NFDiscovery.SearchNFInstancesParamOpts{
		ServiceNames: optional.NewInterface([]string{string(models.ServiceName_NPCF_POLICYAUTHORIZATION)}),
	}
	res, err := s.SearchNFInstances(s.consumer.Config().NrfUri(), models.NfType_PCF, &param)
	if err != nil {
		return "", err
	}

	_, uri, err := getProfileAndUri(res.NfInstances, models.ServiceName_NPCF_POLICYAUTHORIZATION)
	if err != nil {
		logger.ConsumerLog.Errorf(err.Error())
		return "", err
	}
	logger.ConsumerLog.Infof("searchPcfPolicyAuthUri: uri[%s]", uri)

	// TODO: Subscribe NRF to notify service URI change

	return uri, nil
}

func (s *nnrfService) SearchUdrDrUri() (string, error) {
	param := Nnrf_NFDiscovery.SearchNFInstancesParamOpts{
		ServiceNames: optional.NewInterface([]string{string(models.ServiceName_NUDR_DR)}),
	}
	res, err := s.SearchNFInstances(s.consumer.Config().NrfUri(), models.NfType_UDR, &param)
	if err != nil {
		return "", err
	}

	_, uri, err := getProfileAndUri(res.NfInstances, models.ServiceName_NUDR_DR)
	if err != nil {
		logger.ConsumerLog.Errorf(err.Error())
		return "", err
	}
	logger.ConsumerLog.Infof("SearchUdrDrUri: uri[%s]", uri)

	// TODO: Subscribe NRF to notify service URI change

	return uri, nil
}

func getProfileAndUri(nfInstances []models.NfProfile, srvName models.ServiceName) (*models.NfProfile, string, error) {
	// select the first ServiceName
	// TODO: select base on other info
	var profile *models.NfProfile
	var uri string
	for _, nfProfile := range nfInstances {
		profile = &nfProfile
		uri = searchNFServiceUri(nfProfile, srvName, models.NfServiceStatus_REGISTERED)
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
func searchNFServiceUri(nfProfile models.NfProfile, serviceName models.ServiceName,
	nfServiceStatus models.NfServiceStatus) string {
	if nfProfile.NfServices == nil {
		return ""
	}

	nfUri := ""
	for _, service := range *nfProfile.NfServices {
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
			} else if len(*service.IpEndPoints) != 0 {
				// Select the first IpEndPoint
				// TODO: select others when failure
				point := (*service.IpEndPoints)[0]
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
