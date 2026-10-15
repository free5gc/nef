package consumer

import (
	"fmt"
	"sync"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/openapi/nrf/NFDisc"
	"github.com/free5gc/openapi/udr/DR"
	sbi_metrics "github.com/free5gc/util/metrics/sbi"
)

type nudrService struct {
	consumer *Consumer

	mu      sync.RWMutex
	clients map[string]*DR.APIClient
}

func (s *nudrService) getDataRepositoryClient(uri string) *DR.APIClient {
	if uri == "" {
		return nil
	}

	s.mu.RLock()

	client, ok := s.clients[uri]

	if ok {
		defer s.mu.RUnlock()
		return client
	}

	configuration := DR.NewConfiguration()
	configuration.SetBasePath(uri)
	configuration.SetMetrics(sbi_metrics.SbiMetricHook)
	client = DR.NewAPIClient(configuration)

	s.mu.RUnlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[uri] = client
	return client
}

func (s *nudrService) getUdrDrUri() (string, error) {
	uri := s.consumer.Context().UdrDrUri()
	if uri == "" {
		localVarOptionals := NFDisc.SearchNFInstancesRequest{
			ServiceNames: []models.Nrf_NFMgmt_ServiceName{
				models.Nrf_NFMgmt_ServiceName_NUDR_DR,
			},
		}
		_, sUri, err := s.consumer.SearchNFInstances(s.consumer.Config().NrfUri(),
			models.Nrf_NFMgmt_ServiceName_NUDR_DR, models.Nrf_NFMgmt_NFType_UDR,
			models.Nrf_NFMgmt_NFType_NEF, &localVarOptionals)
		if err == nil {
			s.consumer.Context().SetUdrDrUri(sUri)
		}
		return sUri, err
	}
	return uri, nil
}

// AppDataInfluenceDataGet Query the UDR to retrieve models.Udr_DR_TrafficInfluData for each matching combination
// of the values of the elements of the array given in parameters.
// 3GPP TS 29.519 release 17 version 17.6.0
// Resource structure: 6.2.2
// Request/Response: 6.2.5.3.1
func (s *nudrService) AppDataInfluenceDataGet(influenceIDs []string) (
	[]models.Udr_DR_TrafficInfluData, *models.ProblemDetails, error,
) {
	uri, err := s.getUdrDrUri()
	if err != nil {
		return nil, nil, err
	}

	client := s.getDataRepositoryClient(uri)

	if client == nil {
		return nil, nil, fmt.Errorf("could not initialize the DataRepository client")
	}

	param := DR.ReadInfluenceDataRequest{
		InfluenceIds: influenceIDs,
	}

	ctx, _, err := s.consumer.Context().GetTokenCtx(models.Nrf_NFMgmt_ServiceName_NUDR_DR, models.Nrf_NFMgmt_NFType_UDR)
	if err != nil {
		return nil, nil, err
	}

	influenceDataRsp, influenceDataErr := client.InfluenceDataStoreApi.ReadInfluenceData(ctx, &param)

	if influenceDataErr != nil {
		switch apiErr := influenceDataErr.(type) {
		// API error
		case openapi.GenericOpenAPIError:
			switch errorModel := apiErr.Model().(type) {
			case DR.ReadInfluenceDataError:
				return nil, errorModel.ProblemDetails, nil
			case error:
				return nil, openapi.ProblemDetailsSystemFailure(errorModel.Error()), nil
			default:
				return nil, nil, openapi.ReportError("openapi error")
			}
		case error:
			return nil, openapi.ProblemDetailsSystemFailure(apiErr.Error()), nil
		default:
			return nil, nil, openapi.ReportError("server no response")
		}
	}

	return influenceDataRsp.Udr_DR_TrafficInfluData, nil, nil
}

// AppDataInfluenceDataPut Stores the models.Udr_DR_TrafficInfluData for the related influenceID.
// 3GPP TS 29.519 release 17 version 17.6.0
// Resource structure: 6.2.2
// Request/Response: 6.2.6.3.1
func (s *nudrService) AppDataInfluenceDataPut(influenceID string,
	tiData *models.Udr_DR_TrafficInfluData,
) (*models.Udr_DR_TrafficInfluData, *models.ProblemDetails, error) {
	uri, err := s.getUdrDrUri()
	if err != nil {
		return nil, nil, err
	}

	client := s.getDataRepositoryClient(uri)

	if client == nil {
		return nil, nil, openapi.ReportError("could not initialize the DataRepository client")
	}

	ctx, _, err := s.consumer.Context().GetTokenCtx(models.Nrf_NFMgmt_ServiceName_NUDR_DR, models.Nrf_NFMgmt_NFType_UDR)
	if err != nil {
		return nil, nil, err
	}

	influenceDataReq := DR.CreateOrReplaceIndividualInfluenceDataRequest{
		InfluenceId: &influenceID,
		RequestBody: tiData,
	}

	influenceDataResp, errInfluenceData := client.IndividualInfluenceDataDocumentApi.
		CreateOrReplaceIndividualInfluenceData(ctx, &influenceDataReq)

	if errInfluenceData != nil {
		switch apiErr := errInfluenceData.(type) {
		// API error
		case openapi.GenericOpenAPIError:
			switch errorModel := apiErr.Model().(type) {
			case DR.CreateOrReplaceIndividualInfluenceDataError:

				return nil, errorModel.ProblemDetails, nil
			case error:
				return nil, openapi.ProblemDetailsSystemFailure(errorModel.Error()), nil
			default:
				return nil, nil, openapi.ReportError("openapi error")
			}
		case error:
			return nil, openapi.ProblemDetailsSystemFailure(apiErr.Error()), nil
		default:
			return nil, nil, openapi.ReportError("server no response")
		}
	}

	return influenceDataResp.Udr_DR_TrafficInfluData, nil, nil
}

// AppDataPfdsGet Retrieve PFDs for related application identifier(s).
// 3GPP TS 29.519 release 17 version 17.6.0
// Resource structure: 6.2.2
// Request/Response: 6.2.3.3.1
func (s *nudrService) AppDataPfdsGet(
	appIDs []string,
) ([]models.Udr_DR_PfdDataForAppExt, *models.ProblemDetails, error) {
	uri, err := s.getUdrDrUri()
	if err != nil {
		return nil, nil, err
	}

	client := s.getDataRepositoryClient(uri)

	if client == nil {
		return nil, nil, openapi.ReportError("could not initialize the DataRepository client")
	}

	ctx, _, err := s.consumer.Context().GetTokenCtx(models.Nrf_NFMgmt_ServiceName_NUDR_DR, models.Nrf_NFMgmt_NFType_UDR)
	if err != nil {
		return nil, nil, err
	}

	pfdDataReq := DR.ReadPFDDataRequest{
		AppId: appIDs,
	}

	var pfdDataResp *DR.ReadPFDDataResponse
	var errPfdData error
	func() {
		defer func() {
			if p := recover(); p != nil {
				errPfdData = fmt.Errorf("panic from UDR ReadPFDData: %v", p)
			}
		}()

		pfdDataResp, errPfdData = client.PFDDataStoreApi.ReadPFDData(ctx, &pfdDataReq)
	}()

	if errPfdData != nil {
		switch apiErr := errPfdData.(type) {
		// API error
		case openapi.GenericOpenAPIError:
			switch errorModel := apiErr.Model().(type) {
			case DR.ReadPFDDataError:
				return nil, errorModel.ProblemDetails, nil
			case error:
				return nil, openapi.ProblemDetailsSystemFailure(errorModel.Error()), nil
			default:
				return nil, nil, openapi.ReportError("openapi error")
			}
		case error:
			return nil, openapi.ProblemDetailsSystemFailure(apiErr.Error()), nil
		default:
			return nil, nil, openapi.ReportError("server no response")
		}
	}

	if pfdDataResp == nil {
		return nil, openapi.ProblemDetailsSystemFailure("server no response"), nil
	}

	return pfdDataResp.Udr_DR_PfdDataForAppExt, nil, nil
}

// AppDataPfdsAppIdPut Creates, updates an individual PFD given an appId and the content to store into the UDR.
// 3GPP TS 29.519 release 17 version 17.6.0
// Resource structure: 6.2.2
// Request/Response: 6.2.4.3.3
func (s *nudrService) AppDataPfdsAppIdPut(appID string, pfdDataForApp *models.Udr_DR_PfdDataForAppExt) (
	*models.Udr_DR_PfdDataForAppExt, *models.ProblemDetails, error,
) {
	uri, err := s.getUdrDrUri()
	if err != nil {
		return nil, nil, err
	}

	client := s.getDataRepositoryClient(uri)

	if client == nil {
		return nil, nil, openapi.ReportError("could not initialize the DataRepository client")
	}

	ctx, _, err := s.consumer.Context().GetTokenCtx(models.Nrf_NFMgmt_ServiceName_NUDR_DR, models.Nrf_NFMgmt_NFType_UDR)
	if err != nil {
		return nil, nil, err
	}

	individualPfdDataReq := DR.CreateOrReplaceIndividualPFDDataRequest{
		AppId:       &appID,
		RequestBody: pfdDataForApp,
	}

	individualPfdDataRsp, errIndividualPfdData := client.IndividualPFDDataDocumentApi.
		CreateOrReplaceIndividualPFDData(ctx, &individualPfdDataReq)

	if errIndividualPfdData != nil {
		switch apiErr := errIndividualPfdData.(type) {
		// API error
		case openapi.GenericOpenAPIError:
			switch errorModel := apiErr.Model().(type) {
			case DR.CreateOrReplaceIndividualPFDDataError:
				return nil, errorModel.ProblemDetails, nil
			case error:
				return nil, openapi.ProblemDetailsSystemFailure(errorModel.Error()), nil
			default:
				return nil, nil, openapi.ReportError("openapi error")
			}
		case error:
			return nil, openapi.ProblemDetailsSystemFailure(apiErr.Error()), nil
		default:
			return nil, nil, openapi.ReportError("server no response")
		}
	}

	return individualPfdDataRsp.Udr_DR_PfdDataForAppExt, nil, nil
}

// AppDataPfdsAppIdDelete Deletes the individual PFD Data resource related to the application identifier.
// 3GPP TS 29.519 release 17 version 17.6.0
// Resource structure: 6.2.2
// Request/Response: 6.2.4.3.2
func (s *nudrService) AppDataPfdsAppIdDelete(appID string) (*models.ProblemDetails, error) {
	uri, err := s.getUdrDrUri()
	if err != nil {
		return nil, err
	}

	client := s.getDataRepositoryClient(uri)

	if client == nil {
		return nil, openapi.ReportError("could not initialize the DataRepository client")
	}

	ctx, _, err := s.consumer.Context().GetTokenCtx(models.Nrf_NFMgmt_ServiceName_NUDR_DR, models.Nrf_NFMgmt_NFType_UDR)
	if err != nil {
		return nil, err
	}

	DeletePdfDataReq := DR.DeleteIndividualPFDDataRequest{
		AppId: &appID,
	}

	_, errDeletePfdData := client.IndividualPFDDataDocumentApi.DeleteIndividualPFDData(ctx, &DeletePdfDataReq)

	if errDeletePfdData != nil {
		switch apiErr := errDeletePfdData.(type) {
		// API error
		case openapi.GenericOpenAPIError:
			switch errorModel := apiErr.Model().(type) {
			case DR.DeleteIndividualPFDDataError:
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

// AppDataPfdsAppIdGet Returns a representation of PFDs for the related applicationID.
// 3GPP TS 29.519 release 17 version 17.6.0
// Resource structure: 6.2.2
// Request/Response: 6.2.4.3.1
func (s *nudrService) AppDataPfdsAppIdGet(appID string) (
	*DR.ReadIndividualPFDDataResponse, *models.ProblemDetails, error,
) {
	uri, err := s.getUdrDrUri()
	if err != nil {
		return nil, nil, err
	}
	client := s.getDataRepositoryClient(uri)

	if client == nil {
		return nil, nil, openapi.ReportError("could not initialize the DataRepository client")
	}

	ctx, _, err := s.consumer.Context().GetTokenCtx(models.Nrf_NFMgmt_ServiceName_NUDR_DR, models.Nrf_NFMgmt_NFType_UDR)
	if err != nil {
		return nil, nil, err
	}

	pfdDataReq := DR.ReadIndividualPFDDataRequest{
		AppId: &appID,
	}

	pfdData, errPfdData := client.IndividualPFDDataDocumentApi.ReadIndividualPFDData(ctx, &pfdDataReq)

	if errPfdData != nil {
		switch apiErr := errPfdData.(type) {
		// API error
		case openapi.GenericOpenAPIError:
			switch errorModel := apiErr.Model().(type) {
			case DR.ReadIndividualPFDDataError:
				return nil, errorModel.ProblemDetails, nil
			case error:
				return nil, openapi.ProblemDetailsSystemFailure(errorModel.Error()), nil
			default:
				return nil, nil, openapi.ReportError("openapi error")
			}
		case error:
			return nil, openapi.ProblemDetailsSystemFailure(apiErr.Error()), nil
		default:
			return nil, nil, openapi.ReportError("server no response")
		}
	}
	return pfdData, nil, nil
}

// AppDataInfluenceDataPatch Patch the TrafficInfluData for the related influenceID and tiSubPatch and returns it.
// 3GPP TS 29.519 release 17 version 17.6.0
// Resource structure: 6.2.2
// Request/Response: 6.2.6.3.2
func (s *nudrService) AppDataInfluenceDataPatch(
	influenceID string, tiSubPatch *models.Udr_DR_TrafficInfluDataPatch,
) (*models.Udr_DR_TrafficInfluData, *models.ProblemDetails, error) {
	uri, err := s.getUdrDrUri()
	if err != nil {
		return nil, nil, err
	}
	client := s.getDataRepositoryClient(uri)

	ctx, _, err := s.consumer.Context().GetTokenCtx(models.Nrf_NFMgmt_ServiceName_NUDR_DR, models.Nrf_NFMgmt_NFType_UDR)
	if err != nil {
		return nil, nil, err
	}

	tiDataReq := DR.UpdateIndividualInfluenceDataRequest{
		InfluenceId: &influenceID,
		RequestBody: tiSubPatch,
	}

	trafficDataRsp, errTiData := client.IndividualInfluenceDataDocumentApi.UpdateIndividualInfluenceData(ctx, &tiDataReq)

	if errTiData != nil {
		switch apiErr := errTiData.(type) {
		// API error
		case openapi.GenericOpenAPIError:
			switch errorModel := apiErr.Model().(type) {
			case DR.UpdateIndividualInfluenceDataError:
				return nil, errorModel.ProblemDetails, nil
			case error:
				return nil, openapi.ProblemDetailsSystemFailure(errorModel.Error()), nil
			default:
				return nil, nil, openapi.ReportError("openapi error")
			}
		case error:
			return nil, openapi.ProblemDetailsSystemFailure(apiErr.Error()), nil
		default:
			return nil, nil, openapi.ReportError("server no response")
		}
	}

	var trafficInfluData *models.Udr_DR_TrafficInfluData

	if trafficDataRsp != nil {
		trafficInfluData = trafficDataRsp.Udr_DR_TrafficInfluData
	}

	return trafficInfluData, nil, nil
}

// AppDataInfluenceDataDelete Deletes the TrafficInfluenceData for the related influenceID.
// 3GPP TS 29.519 release 17 version 17.6.0
// Resource structure: 6.2.2
// Request/Response: 6.2.6.3.3
func (s *nudrService) AppDataInfluenceDataDelete(influenceID string) (*models.ProblemDetails, error) {
	uri, err := s.getUdrDrUri()
	if err != nil {
		return nil, err
	}
	client := s.getDataRepositoryClient(uri)

	if client == nil {
		return nil, openapi.ReportError("could not initialize the DataRepository client")
	}

	ctx, _, err := s.consumer.Context().GetTokenCtx(models.Nrf_NFMgmt_ServiceName_NUDR_DR, models.Nrf_NFMgmt_NFType_UDR)
	if err != nil {
		return nil, err
	}

	deleteInfluenceReq := DR.DeleteIndividualInfluenceDataRequest{
		InfluenceId: &influenceID,
	}

	_, errDeleteInfluenceData := client.IndividualInfluenceDataDocumentApi.
		DeleteIndividualInfluenceData(ctx, &deleteInfluenceReq)

	if errDeleteInfluenceData != nil {
		switch apiErr := errDeleteInfluenceData.(type) {
		// API error
		case openapi.GenericOpenAPIError:
			switch errorModel := apiErr.Model().(type) {
			case DR.DeleteIndividualInfluenceDataError:
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
