package context

import (
	"slices"

	"github.com/free5gc/openapi/models"
)

const ServiceNameNnefCallback models.Nrf_NFMgmt_ServiceName = "nnef-callback"

var servicePolicies = map[models.Nrf_NFMgmt_ServiceName][]models.Nrf_NFMgmt_NFType{
	models.Nrf_NFMgmt_ServiceName_NNEF_PFDMANAGEMENT: {
		models.Nrf_NFMgmt_NFType_AF,
	},
	models.Nrf_NFMgmt_ServiceName_3GPP_TRAFFIC_INFLUENCE: {
		models.Nrf_NFMgmt_NFType_AF,
	},
	// TS 29.122 northbound API, consumed by AF/AS.
	models.Nrf_NFMgmt_ServiceName_3GPP_MONITORING_EVENT: {
		models.Nrf_NFMgmt_NFType_AF,
	},
	// Inbound event notifications: SMF (Nsmf_EventExposure) and AMF
	// (Namf_EventExposure, used by Monitoring Event).
	ServiceNameNnefCallback: {
		models.Nrf_NFMgmt_NFType_SMF,
		models.Nrf_NFMgmt_NFType_AMF,
	},
	// OAM authorization is intentionally unchanged until management-plane
	// authentication is designed separately.
	models.Nrf_NFMgmt_ServiceName_NNEF_OAM: nil,
}

func AllowedNfTypesForService(serviceName models.Nrf_NFMgmt_ServiceName) (
	[]models.Nrf_NFMgmt_NFType, bool,
) {
	allowed, known := servicePolicies[serviceName]
	return slices.Clone(allowed), known
}
