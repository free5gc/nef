package consumer

import (
	"net/http"
	"sync"

	"github.com/free5gc/nef/internal/logger"
	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/openapi/nrf/NFDisc"
	"github.com/free5gc/openapi/udm/SDM"
	sbi_metrics "github.com/free5gc/util/metrics/sbi"
)

type nudmService struct {
	consumer *Consumer

	mu      sync.RWMutex
	clients map[string]*SDM.APIClient
}

func (s *nudmService) getSdmClient(uri string) *SDM.APIClient {
	if uri == "" {
		return nil
	}

	s.mu.RLock()
	if client, ok := s.clients[uri]; ok {
		defer s.mu.RUnlock()
		return client
	}
	s.mu.RUnlock()

	configuration := SDM.NewConfiguration()
	configuration.SetBasePath(uri)
	configuration.SetMetrics(sbi_metrics.SbiMetricHook)
	configuration.SetHTTPClient(http.DefaultClient)
	cli := SDM.NewAPIClient(configuration)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[uri] = cli
	return cli
}

func (s *nudmService) getUdmSdmUri() (string, error) {
	uri := s.consumer.Context().UdmSdmUri()
	if uri == "" {
		localVarOptionals := NFDisc.SearchNFInstancesRequest{
			ServiceNames: []models.Nrf_NFMgmt_ServiceName{
				models.Nrf_NFMgmt_ServiceName_NUDM_SDM,
			},
		}
		_, sUri, err := s.consumer.SearchNFInstances(
			s.consumer.Config().NrfUri(),
			models.Nrf_NFMgmt_ServiceName_NUDM_SDM,
			models.Nrf_NFMgmt_NFType_UDM,
			models.Nrf_NFMgmt_NFType_NEF,
			&localVarOptionals,
		)
		if err == nil {
			s.consumer.Context().SetUdmSdmUri(sUri)
		}
		return sUri, err
	}
	return uri, nil
}

// GetSupiFromGpsi resolves an AF-facing GPSI (external ID/MSISDN) to the SUPI AMF's
// Namf_EventExposure subscription actually requires, via Nudm_SDM's GPSI-to-SUPI
// translation (3GPP TS 29.503 clause 5.2.2.7.1), which itself queries UDR.
func (s *nudmService) GetSupiFromGpsi(gpsi string) (supi string, pd *models.ProblemDetails, err error) {
	uri, err := s.getUdmSdmUri()
	if err != nil {
		return "", nil, err
	}

	client := s.getSdmClient(uri)
	if client == nil {
		return "", nil, openapi.ReportError("could not initialize the SubscriberDataManagement client")
	}

	ctx, _, err := s.consumer.Context().GetTokenCtx(
		models.Nrf_NFMgmt_ServiceName_NUDM_SDM, models.Nrf_NFMgmt_NFType_UDM)
	if err != nil {
		return "", nil, err
	}

	req := &SDM.GetSupiOrGpsiRequest{
		UeId: &gpsi,
	}

	rsp, errGetSupiOrGpsi := client.GPSIToSUPITranslationOrSUPIToGPSITranslationApi.GetSupiOrGpsi(ctx, req)
	if errGetSupiOrGpsi != nil {
		switch apiErr := errGetSupiOrGpsi.(type) {
		case openapi.GenericOpenAPIError:
			switch errorModel := apiErr.Model().(type) {
			case SDM.GetSupiOrGpsiError:
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

	logger.ConsumerLog.Debugf("GetSupiFromGpsi RspData: %+v", rsp.Udm_SDM_IdTranslationResult)
	if rsp.Udm_SDM_IdTranslationResult.Supi == "" {
		return "", openapi.ProblemDetailsDataNotFound("No SUPI found for the given GPSI"), nil
	}
	return rsp.Udm_SDM_IdTranslationResult.Supi, nil, nil
}
