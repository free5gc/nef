package consumer

import (
	"net/http"
	"sync"

	"github.com/free5gc/nef/internal/logger"
	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/amf/EvtExpos"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/openapi/nrf/NFDisc"
	sbi_metrics "github.com/free5gc/util/metrics/sbi"
)

type namfService struct {
	consumer *Consumer

	mu      sync.RWMutex
	clients map[string]*EvtExpos.APIClient
}

func (s *namfService) getEventExposureClient(uri string) *EvtExpos.APIClient {
	if uri == "" {
		return nil
	}

	s.mu.RLock()
	if client, ok := s.clients[uri]; ok {
		defer s.mu.RUnlock()
		return client
	}
	s.mu.RUnlock()

	configuration := EvtExpos.NewConfiguration()
	configuration.SetBasePath(uri)
	configuration.SetMetrics(sbi_metrics.SbiMetricHook)
	configuration.SetHTTPClient(http.DefaultClient)
	cli := EvtExpos.NewAPIClient(configuration)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[uri] = cli
	return cli
}

func (s *namfService) getNamfEvtsUri() (string, error) {
	uri := s.consumer.Context().NamfEvtsUri()
	if uri == "" {
		localVarOptionals := NFDisc.SearchNFInstancesRequest{
			ServiceNames: []models.Nrf_NFMgmt_ServiceName{
				models.Nrf_NFMgmt_ServiceName_NAMF_EVTS,
			},
		}
		_, sUri, err := s.consumer.SearchNFInstances(
			s.consumer.Config().NrfUri(),
			models.Nrf_NFMgmt_ServiceName_NAMF_EVTS,
			models.Nrf_NFMgmt_NFType_AMF,
			models.Nrf_NFMgmt_NFType_NEF,
			&localVarOptionals,
		)
		if err == nil {
			s.consumer.Context().SetNamfEvtsUri(sUri)
		}
		return sUri, err
	}
	return uri, nil
}

// CreateEventSubscription creates a Namf_EventExposure subscription on behalf of an AF's
// monitoring event subscription. AMF's CreateAMFEventSubscriptionProcedure looks up the
// target UE by Supi only (it does not resolve Gpsi), so the caller must already have
// translated the AF-facing GPSI to a SUPI (see nudmService.GetSupiFromGpsi).
// 3GPP TS 29.518 Release 17
func (s *namfService) CreateEventSubscription(
	supi string,
	events []models.Amf_EvtExpos_AmfEventType,
	notifyUri, correlationID string,
) (amfSubID string, pd *models.ProblemDetails, err error) {
	uri, err := s.getNamfEvtsUri()
	if err != nil {
		return "", nil, err
	}

	client := s.getEventExposureClient(uri)
	if client == nil {
		return "", nil, openapi.ReportError("could not initialize the EventExposure client")
	}

	ctx, _, err := s.consumer.Context().GetTokenCtx(
		models.Nrf_NFMgmt_ServiceName_NAMF_EVTS, models.Nrf_NFMgmt_NFType_AMF)
	if err != nil {
		return "", nil, err
	}

	eventList := make([]models.Amf_EvtExpos_AmfEvent, 0, len(events))
	for _, evtType := range events {
		eventList = append(eventList, models.Amf_EvtExpos_AmfEvent{Type: evtType})
	}

	createReq := &EvtExpos.CreateSubscriptionRequest{
		RequestBody: &models.Amf_EvtExpos_AmfCreateEventSubscription{
			Subscription: &models.Amf_EvtExpos_AmfEventSubscription{
				EventList:           eventList,
				EventNotifyUri:      notifyUri,
				NotifyCorrelationId: correlationID,
				Supi:                supi,
			},
		},
	}

	rsp, errCreateSub := client.SubscriptionsCollectionCollectionApi.CreateSubscription(ctx, createReq)
	if errCreateSub != nil {
		switch apiErr := errCreateSub.(type) {
		case openapi.GenericOpenAPIError:
			switch errorModel := apiErr.Model().(type) {
			case EvtExpos.CreateSubscriptionError:
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

	logger.ConsumerLog.Debugf("CreateEventSubscription RspData: %+v", rsp.Amf_EvtExpos_AmfCreatedEventSubscription)
	return rsp.Amf_EvtExpos_AmfCreatedEventSubscription.SubscriptionId, nil, nil
}

// DeleteEventSubscription deletes a Namf_EventExposure subscription previously created via
// CreateEventSubscription.
// 3GPP TS 29.518 Release 17
func (s *namfService) DeleteEventSubscription(amfSubID string) (*models.ProblemDetails, error) {
	uri, err := s.getNamfEvtsUri()
	if err != nil {
		return nil, err
	}

	client := s.getEventExposureClient(uri)
	if client == nil {
		return nil, openapi.ReportError("could not initialize the EventExposure client")
	}

	ctx, _, err := s.consumer.Context().GetTokenCtx(
		models.Nrf_NFMgmt_ServiceName_NAMF_EVTS, models.Nrf_NFMgmt_NFType_AMF)
	if err != nil {
		return nil, err
	}

	deleteReq := &EvtExpos.DeleteSubscriptionRequest{
		SubscriptionId: &amfSubID,
	}

	_, errDeleteSub := client.IndividualSubscriptionDocumentApi.DeleteSubscription(ctx, deleteReq)
	if errDeleteSub != nil {
		switch apiErr := errDeleteSub.(type) {
		case openapi.GenericOpenAPIError:
			switch errorModel := apiErr.Model().(type) {
			case EvtExpos.DeleteSubscriptionError:
				return errorModel.ProblemDetails, nil
			case error:
				return openapi.ProblemDetailsSystemFailure(errorModel.Error()), nil
			default:
				return nil, openapi.ReportError("openapi error")
			}
		case error:
			return openapi.ProblemDetailsSystemFailure(apiErr.Error()), nil
		default:
			return nil, openapi.ReportError("server no response")
		}
	}

	return nil, nil
}
