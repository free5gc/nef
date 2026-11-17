package processor

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/gin-gonic/gin"
	"github.com/h2non/gock"
	"github.com/stretchr/testify/require"
)

func initNRFDiscBsfMgmtStub() {
	searchResult := &models.Nrf_NFDisc_SearchResult{
		ValidityPeriod: 100,
		NfInstances: []models.Nrf_NFDisc_NFProfile{
			{
				NfInstanceId: "nef-unit-testing",
				NfType:       "BSF",
				NfStatus:     "REGISTERED",
				Ipv4Addresses: []string{
					"127.0.0.6",
				},
				NfServices: []models.Nrf_NFDisc_NFService{
					{
						ServiceInstanceId: "1",
						ServiceName:       "nbsf-management",
						Versions: []models.Nrf_NFMgmt_NFServiceVersion{
							{
								ApiVersionInUri: "v1",
								ApiFullVersion:  "1.0.0",
							},
						},
						Scheme:          "http",
						NfServiceStatus: "REGISTERED",
						IpEndPoints: []models.Nrf_NFMgmt_IpEndPoint{
							{
								Ipv4Address: "127.0.0.6",
								Transport:   "TCP",
								Port:        8000,
							},
						},
						ApiPrefix: "http://127.0.0.6:8000",
					},
				},
			},
		},
	}

	gock.New("http://127.0.0.10:8000/nnrf-disc/v1").
		Get("/nf-instances").
		MatchParam("target-nf-type", "BSF").
		MatchParam("requester-nf-type", "NEF").
		MatchParam("service-names", "nbsf-management").
		Reply(http.StatusOK).
		JSON(searchResult)
}

func initBsfGetPCFBindingsFoundStub(supi string) {
	rsp := models.Bsf_Mgmt_PcfBinding{
		Supi: supi,
	}
	gock.New("http://127.0.0.6:8000/nbsf-management/v1").
		Get("/pcfBindings").
		Reply(http.StatusOK).
		JSON(rsp)
}

func initBsfGetPCFBindingsNotFoundStub() {
	gock.New("http://127.0.0.6:8000/nbsf-management/v1").
		Get("/pcfBindings").
		Reply(http.StatusNoContent)
}

func initUDMSdmGetGpsiStub(statusCode int, gpsi string) {
	rsp := models.Udm_SDM_IdTranslationResult{
		Gpsi: gpsi,
	}
	gock.New("http://127.0.0.3:8000/nudm-sdm/v2").
		Get("/.*/id-translation-result").
		Reply(statusCode).
		JSON(rsp)
}

func TestRetrieveUeId(t *testing.T) {
	testCases := []struct {
		description      string
		req              models.Nef_UEId_UeIdReq
		setupMocks       func()
		expectedResponse *HandlerResponse
	}{
		{
			description: "TC1: valid ueIpAddr, should return externalId",
			req: models.Nef_UEId_UeIdReq{
				AfId:     "af1",
				UeIpAddr: &models.IpAddr{Ipv4Addr: "10.60.0.1"},
			},
			setupMocks: func() {
				initNRFDiscBsfMgmtStub()
				initBsfGetPCFBindingsFoundStub("imsi-208930000000001")
				initNRFDiscUDMSdmStub()
				initUDMSdmGetGpsiStub(http.StatusOK, "extid-ue001@domain.com")
			},
			expectedResponse: &HandlerResponse{
				Status: http.StatusOK,
				Body:   &models.Nef_UEId_UeIdInfo{ExternalId: "ue001@domain.com"},
			},
		},
		{
			description: "TC2: valid ueMacAddr, should return externalId",
			req: models.Nef_UEId_UeIdReq{
				AfId:      "af1",
				UeMacAddr: "aa-bb-cc-dd-ee-ff",
			},
			setupMocks: func() {
				initNRFDiscBsfMgmtStub()
				initBsfGetPCFBindingsFoundStub("imsi-208930000000001")
				initNRFDiscUDMSdmStub()
				initUDMSdmGetGpsiStub(http.StatusOK, "extid-ue001@domain.com")
			},
			expectedResponse: &HandlerResponse{
				Status: http.StatusOK,
				Body:   &models.Nef_UEId_UeIdInfo{ExternalId: "ue001@domain.com"},
			},
		},
		{
			description: "TC3: missing afId, should return 400",
			req: models.Nef_UEId_UeIdReq{
				UeIpAddr: &models.IpAddr{Ipv4Addr: "10.60.0.1"},
			},
			setupMocks: func() {},
			expectedResponse: &HandlerResponse{
				Status: http.StatusBadRequest,
				Body: &models.ProblemDetails{
					Status: http.StatusBadRequest,
					Title:  "Malformed request syntax",
					Detail: "Missing afId",
				},
			},
		},
		{
			description: "TC4: neither ueIpAddr nor ueMacAddr, should return 400",
			req: models.Nef_UEId_UeIdReq{
				AfId: "af1",
			},
			setupMocks: func() {},
			expectedResponse: &HandlerResponse{
				Status: http.StatusBadRequest,
				Body: &models.ProblemDetails{
					Status: http.StatusBadRequest,
					Title:  "Malformed request syntax",
					Detail: "Exactly one of ueIpAddr, ueMacAddr must be present",
				},
			},
		},
		{
			description: "TC5: both ueIpAddr and ueMacAddr, should return 400",
			req: models.Nef_UEId_UeIdReq{
				AfId:      "af1",
				UeIpAddr:  &models.IpAddr{Ipv4Addr: "10.60.0.1"},
				UeMacAddr: "aa-bb-cc-dd-ee-ff",
			},
			setupMocks: func() {},
			expectedResponse: &HandlerResponse{
				Status: http.StatusBadRequest,
				Body: &models.ProblemDetails{
					Status: http.StatusBadRequest,
					Title:  "Malformed request syntax",
					Detail: "Exactly one of ueIpAddr, ueMacAddr must be present",
				},
			},
		},
		{
			description: "TC6: IPv6-only ueIpAddr, should return 400",
			req: models.Nef_UEId_UeIdReq{
				AfId:     "af1",
				UeIpAddr: &models.IpAddr{Ipv6Addr: "2001:db8::1"},
			},
			setupMocks: func() {},
			expectedResponse: &HandlerResponse{
				Status: http.StatusBadRequest,
				Body: &models.ProblemDetails{
					Status: http.StatusBadRequest,
					Title:  "Malformed request syntax",
					Detail: "IPv6-keyed ueIpAddr is not yet supported; provide ueIpAddr.ipv4Addr or ueMacAddr",
				},
			},
		},
		{
			description: "TC7: no PCF binding found for the given address, should return 404",
			req: models.Nef_UEId_UeIdReq{
				AfId:     "af1",
				UeIpAddr: &models.IpAddr{Ipv4Addr: "10.60.0.99"},
			},
			setupMocks: func() {
				initNRFDiscBsfMgmtStub()
				initBsfGetPCFBindingsNotFoundStub()
			},
			expectedResponse: &HandlerResponse{
				Status: http.StatusNotFound,
				Body: &models.ProblemDetails{
					Status: http.StatusNotFound,
					Title:  "Data not found",
					Detail: "No PCF binding found for the given UE address",
				},
			},
		},
		{
			description: "TC8: UE has only an MSISDN-formatted Gpsi, no externalId, should return 404",
			req: models.Nef_UEId_UeIdReq{
				AfId:     "af1",
				UeIpAddr: &models.IpAddr{Ipv4Addr: "10.60.0.1"},
			},
			setupMocks: func() {
				initNRFDiscBsfMgmtStub()
				initBsfGetPCFBindingsFoundStub("imsi-208930000000001")
				initNRFDiscUDMSdmStub()
				initUDMSdmGetGpsiStub(http.StatusOK, "msisdn-0900000001")
			},
			expectedResponse: &HandlerResponse{
				Status: http.StatusNotFound,
				Body: &models.ProblemDetails{
					Status: http.StatusNotFound,
					Title:  "Data not found",
					Detail: "No external identifier found for the given UE",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			openapi.InterceptInnerHttp2Client(t, false)
			tc.setupMocks()
			defer gock.Off()

			httpRecorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(httpRecorder)

			nefApp.Processor().RetrieveUeId(c, &tc.req)
			require.Equal(t, tc.expectedResponse.Status, httpRecorder.Code)

			assertJSONBodyEqual(t, tc.expectedResponse.Body, httpRecorder.Body.Bytes())
		})
	}
}
