package consumer

import (
	ctx "context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/antihax/optional"

	"bitbucket.org/free5gc-team/nef/internal/context"
	"bitbucket.org/free5gc-team/nef/internal/factory"
	"bitbucket.org/free5gc-team/nef/internal/logger"
	"bitbucket.org/free5gc-team/openapi/Nnrf_NFDiscovery"
	"bitbucket.org/free5gc-team/openapi/Nnrf_NFManagement"
	"bitbucket.org/free5gc-team/openapi/models"
)

type ConsumerNRFService struct {
	cfg            *factory.Config
	nefCtx         *context.NefContext
	clientNFMngmnt *Nnrf_NFManagement.APIClient
	clientNFDisc   *Nnrf_NFDiscovery.APIClient
}

func NewConsumerNRFService(nefCfg *factory.Config, nefCtx *context.NefContext) *ConsumerNRFService {
	c := &ConsumerNRFService{cfg: nefCfg, nefCtx: nefCtx}

	nfMngmntConfig := Nnrf_NFManagement.NewConfiguration()
	nfMngmntConfig.SetBasePath(c.cfg.GetNrfUri())
	c.clientNFMngmnt = Nnrf_NFManagement.NewAPIClient(nfMngmntConfig)

	nfDiscConfig := Nnrf_NFDiscovery.NewConfiguration()
	nfDiscConfig.SetBasePath(c.cfg.GetNrfUri())
	c.clientNFDisc = Nnrf_NFDiscovery.NewAPIClient(nfDiscConfig)
	return c
}

func (c *ConsumerNRFService) buildNfProfile(serviceList []factory.Service) *models.NfProfile {
	profile := &models.NfProfile{
		NfInstanceId: c.nefCtx.GetNfInstID(),
		NfType:       models.NfType_NEF,
		NfStatus:     models.NfStatus_REGISTERED,
	}
	profile.Ipv4Addresses = append(profile.Ipv4Addresses, c.cfg.GetSbiRegisterIP())

	versions := strings.Split(c.cfg.GetVersion(), ".")
	majorVersionUri := "v" + versions[0]
	nfServices := []models.NfService{}
	for i, service := range serviceList {
		nfService := models.NfService{
			ServiceInstanceId: strconv.Itoa(i),
			ServiceName:       models.ServiceName(service.ServiceName),
			Versions: &[]models.NfServiceVersion{
				{
					ApiFullVersion:  c.cfg.GetVersion(),
					ApiVersionInUri: majorVersionUri,
				},
			},
			Scheme:          models.UriScheme(c.cfg.GetSbiScheme()),
			NfServiceStatus: models.NfServiceStatus_REGISTERED,
			ApiPrefix:       c.cfg.GetSbiUri(),
			IpEndPoints: &[]models.IpEndPoint{
				{
					Ipv4Address: c.cfg.GetSbiRegisterIP(),
					Transport:   models.TransportProtocol_TCP,
					Port:        int32(c.cfg.GetSbiPort()),
				},
			},
			SupportedFeatures: service.SuppFeat,
		}
		nfServices = append(nfServices, nfService)
	}
	profile.NfServices = &nfServices
	return profile
}

func (c *ConsumerNRFService) RegisterNFInstance() {
	var rsp *http.Response
	var err error

	list := c.cfg.GetServiceList()
	if list == nil {
		logger.ConsumerLog.Warnf("No service to register to NRF")
		return
	}

	for {
		_, rsp, err = c.clientNFMngmnt.NFInstanceIDDocumentApi.RegisterNFInstance(
			ctx.Background(), c.nefCtx.GetNfInstID(), *c.buildNfProfile(list))
		if err != nil || rsp == nil {
			logger.ConsumerLog.Infof("NEF register to NRF Error[%v], sleep 2s and retry", err)
			time.Sleep(2 * time.Second)
			continue
		}
		status := rsp.StatusCode
		if status == http.StatusOK {
			// NFUpdate
			logger.ConsumerLog.Infof("NFRegister Update")
			break
		} else if status == http.StatusCreated {
			// NFRegister
			resrcUri := rsp.Header.Get("Location")
			//resrcNrfUri := resrcUri[:strings.Index(resrcUri, "/nnrf-nfm/")]
			c.nefCtx.NfInstID(resrcUri[strings.LastIndex(resrcUri, "/")+1:])
			logger.ConsumerLog.Infof("NFRegister Created")
			break
		} else {
			logger.ConsumerLog.Infof("NRF return wrong status: %d", status)
		}
	}
}

func (c *ConsumerNRFService) SearchNFServiceUri(targetNfType string, srvName string) (string, error) {
	param := Nnrf_NFDiscovery.SearchNFInstancesParamOpts{
		ServiceNames: optional.NewInterface([]models.ServiceName{models.ServiceName(srvName)}),
	}
	result, rsp, err := c.clientNFDisc.NFInstancesStoreApi.SearchNFInstances(ctx.Background(),
		models.NfType(targetNfType), models.NfType_NEF, &param)
	if rsp != nil && rsp.StatusCode == http.StatusTemporaryRedirect {
		err = fmt.Errorf("SearchNFInstance Error: Temporary Redirect")
	}
	if err != nil {
		return "", fmt.Errorf("SearchNFInstance Error: %+v", err)
	}

	uri := ""
	for _, nfProfile := range result.NfInstances {
		if uri = searchUriFromNfProfile(nfProfile, models.ServiceName(srvName),
			models.NfServiceStatus_REGISTERED); uri != "" {
			break
		}
	}
	if uri == "" {
		err = fmt.Errorf("SearchNFServiceUri Error: no URI found")
		return "", err
	}
	return uri, nil
}

// searchUriFromNfProfile returns NF Uri derived from NfProfile with corresponding service
func searchUriFromNfProfile(nfProfile models.NfProfile, serviceName models.ServiceName,
	nfServiceStatus models.NfServiceStatus) string {
	nfUri := ""
	if nfProfile.NfServices != nil {
		for _, service := range *nfProfile.NfServices {
			if service.ServiceName == serviceName && service.NfServiceStatus == nfServiceStatus {
				if nfProfile.Fqdn != "" {
					nfUri = nfProfile.Fqdn
				} else if service.Fqdn != "" {
					nfUri = service.Fqdn
				} else if service.ApiPrefix != "" {
					nfUri = service.ApiPrefix
				} else if service.IpEndPoints != nil {
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
