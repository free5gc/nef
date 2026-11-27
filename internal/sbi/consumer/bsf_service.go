package consumer

import (
	"net/http"
	"sync"

	"github.com/free5gc/nef/internal/logger"
	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/bsf/Mgmt"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/openapi/nrf/NFDisc"
	sbi_metrics "github.com/free5gc/util/metrics/sbi"
)

type nbsfService struct {
	consumer *Consumer

	mu      sync.RWMutex
	clients map[string]*Mgmt.APIClient
}

func (s *nbsfService) getMgmtClient(uri string) *Mgmt.APIClient {
	if uri == "" {
		return nil
	}

	s.mu.RLock()
	if client, ok := s.clients[uri]; ok {
		defer s.mu.RUnlock()
		return client
	}
	s.mu.RUnlock()

	configuration := Mgmt.NewConfiguration()
	configuration.SetBasePath(uri)
	configuration.SetMetrics(sbi_metrics.SbiMetricHook)
	configuration.SetHTTPClient(http.DefaultClient)
	cli := Mgmt.NewAPIClient(configuration)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[uri] = cli
	return cli
}

func (s *nbsfService) getBsfMgmtUri() (string, error) {
	uri := s.consumer.Context().BsfMgmtUri()
	if uri == "" {
		localVarOptionals := NFDisc.SearchNFInstancesRequest{
			ServiceNames: []models.Nrf_NFMgmt_ServiceName{
				models.Nrf_NFMgmt_ServiceName_NBSF_MANAGEMENT,
			},
		}
		_, sUri, err := s.consumer.SearchNFInstances(
			s.consumer.Config().NrfUri(),
			models.Nrf_NFMgmt_ServiceName_NBSF_MANAGEMENT,
			models.Nrf_NFMgmt_NFType_BSF,
			models.Nrf_NFMgmt_NFType_NEF,
			&localVarOptionals,
		)
		if err == nil {
			s.consumer.Context().SetBsfMgmtUri(sUri)
		}
		return sUri, err
	}
	return uri, nil
}

// GetSupiFromUeAddr resolves a UE's IP or MAC address to its SUPI via BSF's
// Nbsf_Management_Discovery service (3GPP TS 29.521 clause 5.2), which looks up the
// PCF binding currently serving that address. Exactly one of ipv4Addr/ipv6Prefix/macAddr48
// should be set; dnn narrows the lookup when a UE has sessions on multiple DNNs.
// Returns ("", nil, nil) if no binding is currently found for the given address (BSF 204).
func (s *nbsfService) GetSupiFromUeAddr(
	ipv4Addr, macAddr48, dnn string,
) (supi string, pd *models.ProblemDetails, err error) {
	uri, err := s.getBsfMgmtUri()
	if err != nil {
		return "", nil, err
	}

	client := s.getMgmtClient(uri)
	if client == nil {
		return "", nil, openapi.ReportError("could not initialize the Nbsf_Management client")
	}

	ctx, _, err := s.consumer.Context().GetTokenCtx(
		models.Nrf_NFMgmt_ServiceName_NBSF_MANAGEMENT, models.Nrf_NFMgmt_NFType_BSF)
	if err != nil {
		return "", nil, err
	}

	req := &Mgmt.GetPCFBindingsRequest{}
	if ipv4Addr != "" {
		req.SetIpv4Addr(ipv4Addr)
	}
	if macAddr48 != "" {
		req.SetMacAddr48(macAddr48)
	}
	if dnn != "" {
		req.SetDnn(dnn)
	}

	rsp, errGetBindings := client.PCFBindingsCollectionApi.GetPCFBindings(ctx, req)
	if errGetBindings != nil {
		switch apiErr := errGetBindings.(type) {
		case openapi.GenericOpenAPIError:
			switch errorModel := apiErr.Model().(type) {
			case Mgmt.GetPCFBindingsError:
				return "", errorModel.ProblemDetails, nil
			case error:
				return "", openapi.ProblemDetailsSystemFailure(errorModel.Error()), nil
			default:
				return "", nil, openapi.ReportError("openapi error")
			}
		case error:
			return "", openapi.ProblemDetailsSystemFailure(apiErr.Error()), nil
		default:
			return "", nil, openapi.ReportError("server no response")
		}
	}

	if rsp.Bsf_Mgmt_PcfBinding == nil {
		// BSF 204: no PCF binding currently matches this address.
		return "", nil, nil
	}

	logger.ConsumerLog.Debugf("GetSupiFromUeAddr RspData: %+v", rsp.Bsf_Mgmt_PcfBinding)
	return rsp.Bsf_Mgmt_PcfBinding.Supi, nil, nil
}
