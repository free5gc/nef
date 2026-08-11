package processor

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/free5gc/openapi/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
)

var (
	monSub1ForAf1 = models.NefMonitoringEventSubscription{
		ExternalId:              "10001@domain.com",
		NotificationDestination: "http://127.0.0.100:8000/nnef-callback/v1/monitoring-event/app1",
		MonitoringType:          models.MonitoringType_UE_REACHABILITY,
		MaximumNumberOfReports:  1,
	}

	monSub2ForAf1 = models.NefMonitoringEventSubscription{
		ExternalId:             "10002@domain.com",
		MonitoringType:         models.MonitoringType_UE_REACHABILITY,
		MaximumNumberOfReports: 1,
		// NotificationDestination intentionally missing
	}

	monSub3ForAf1 = models.NefMonitoringEventSubscription{
		ExternalId:              "10003@domain.com",
		NotificationDestination: "http://127.0.0.100:8000/nnef-callback/v1/monitoring-event/app3",
		MonitoringType:          models.MonitoringType_ROAMING_STATUS, // not yet supported
		MaximumNumberOfReports:  1,
	}

	monSub4ForAf1 = models.NefMonitoringEventSubscription{
		Ipv4Addr:                "10.60.0.10", // no ExternalId/Msisdn
		NotificationDestination: "http://127.0.0.100:8000/nnef-callback/v1/monitoring-event/app4",
		MonitoringType:          models.MonitoringType_LOCATION_REPORTING,
		MaximumNumberOfReports:  1,
	}

	monSub5ForAf1 = models.NefMonitoringEventSubscription{
		ExternalId:              "10005@domain.com",
		NotificationDestination: "http://127.0.0.100:8000/nnef-callback/v1/monitoring-event/app5",
		MonitoringType:          models.MonitoringType_LOSS_OF_CONNECTIVITY,
		// MaximumNumberOfReports and MonitorExpireTime both missing
	}
)

func TestGetMonitoringEventSubscriptions(t *testing.T) {
	testCases := []struct {
		description      string
		afID             string
		expectedResponse *HandlerResponse
	}{
		{
			description: "TC1: AfID found, should return all monitoring event subscriptions",
			afID:        "af1",
			expectedResponse: &HandlerResponse{
				Status: http.StatusOK,
				Body:   &[]models.NefMonitoringEventSubscription{monSub1ForAf1},
			},
		},
		{
			description: "TC2: AfID not found, should return ProblemDetails",
			afID:        "af3",
			expectedResponse: &HandlerResponse{
				Status: http.StatusNotFound,
				Body: &models.ProblemDetails{
					Status: http.StatusNotFound,
					Title:  "Data not found",
					Detail: "AF is not found",
				},
			},
		},
	}

	nefCtx := nefApp.Context()
	af1 := nefCtx.NewAf("af1")
	af1.Mu.Lock()
	correID1 := nefCtx.NewCorreID()
	monSubCtx1 := af1.NewMonSub(correID1, &monSub1ForAf1)
	af1.MonSubs[monSubCtx1.SubID] = monSubCtx1
	nefCtx.AddAf(af1)
	af1.Mu.Unlock()

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			httpRecorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(httpRecorder)

			nefApp.Processor().GetMonitoringEventSubscriptions(c, tc.afID)
			require.Equal(t, tc.expectedResponse.Status, httpRecorder.Code)

			if monSubs, ok := tc.expectedResponse.Body.(*[]models.NefMonitoringEventSubscription); ok {
				var rspSubs []models.NefMonitoringEventSubscription
				require.NoError(t, json.Unmarshal(httpRecorder.Body.Bytes(), &rspSubs))
				require.ElementsMatch(t, *monSubs, rspSubs)
			} else {
				assertJSONBodyEqual(t, tc.expectedResponse.Body, httpRecorder.Body.Bytes())
			}
		})
	}

	nefCtx.DeleteAf(af1.AfID)
	nefCtx.ResetCorreID()
}

func TestGetIndividualMonitoringEventSubscription(t *testing.T) {
	testCases := []struct {
		description      string
		afID             string
		subID            string
		expectedResponse *HandlerResponse
	}{
		{
			description: "TC1: AfID & SubID found, should return the subscription",
			afID:        "af1",
			subID:       "1",
			expectedResponse: &HandlerResponse{
				Status: http.StatusOK,
				Body:   &monSub1ForAf1,
			},
		},
		{
			description: "TC2: AfID found but SubID not found, should return ProblemDetails",
			afID:        "af1",
			subID:       "2",
			expectedResponse: &HandlerResponse{
				Status: http.StatusNotFound,
				Body: &models.ProblemDetails{
					Status: http.StatusNotFound,
					Title:  "Data not found",
					Detail: "Subscription is not found",
				},
			},
		},
		{
			description: "TC3: AfID not found, should return ProblemDetails",
			afID:        "af3",
			subID:       "1",
			expectedResponse: &HandlerResponse{
				Status: http.StatusNotFound,
				Body: &models.ProblemDetails{
					Status: http.StatusNotFound,
					Title:  "Data not found",
					Detail: "AF is not found",
				},
			},
		},
	}

	nefCtx := nefApp.Context()
	af1 := nefCtx.NewAf("af1")
	af1.Mu.Lock()
	correID1 := nefCtx.NewCorreID()
	monSubCtx1 := af1.NewMonSub(correID1, &monSub1ForAf1)
	af1.MonSubs[monSubCtx1.SubID] = monSubCtx1
	nefCtx.AddAf(af1)
	af1.Mu.Unlock()

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			httpRecorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(httpRecorder)

			nefApp.Processor().GetIndividualMonitoringEventSubscription(c, tc.afID, tc.subID)
			require.Equal(t, tc.expectedResponse.Status, httpRecorder.Code)

			assertJSONBodyEqual(t, tc.expectedResponse.Body, httpRecorder.Body.Bytes())
		})
	}

	nefCtx.DeleteAf(af1.AfID)
	nefCtx.ResetCorreID()
}

func TestPostMonitoringEventSubscription(t *testing.T) {
	// Mocks below are intentionally not gock.Off()'d: Off() flushes gock's global mock
	// registry, including the one-shot NRF discovery mocks other test files register once
	// in TestMain, breaking them for whichever test runs later. Each mock here is scoped
	// to a single expected call and self-disables after matching, so no cleanup is needed.
	initNRFDiscUDMSdmStub()
	initUDMSdmGetSupiOrGpsiStub(http.StatusOK, "imsi-208930000000001")
	initNRFDiscAMFStub()
	initAMFEvtsCreateSubscriptionStub(http.StatusCreated, "amf-sub-1")

	rspMonSub1 := monSub1ForAf1
	rspMonSub1.Self = nefApp.Processor().genMonitoringEventSubURI("af1", "1")

	testCases := []struct {
		description      string
		afID             string
		monSub           *models.NefMonitoringEventSubscription
		expectedResponse *HandlerResponse
	}{
		{
			description: "TC1: Successful UE_REACHABILITY subscription, should create a Namf_EventExposure subscription",
			afID:        "af1",
			monSub:      &monSub1ForAf1,
			expectedResponse: &HandlerResponse{
				Status: http.StatusCreated,
				Headers: map[string][]string{
					"Location": {rspMonSub1.Self},
				},
				Body: &rspMonSub1,
			},
		},
		{
			description: "TC2: Missing notificationDestination",
			afID:        "af1",
			monSub:      &monSub2ForAf1,
			expectedResponse: &HandlerResponse{
				Status: http.StatusBadRequest,
				Body: &models.ProblemDetails{
					Status: http.StatusBadRequest,
					Title:  "Malformed request syntax",
					Detail: "Missing notificationDestination",
				},
			},
		},
		{
			description: "TC3: Unsupported monitoringType",
			afID:        "af1",
			monSub:      &monSub3ForAf1,
			expectedResponse: &HandlerResponse{
				Status: http.StatusBadRequest,
				Body: &models.ProblemDetails{
					Status: http.StatusBadRequest,
					Title:  "Malformed request syntax",
					Detail: "Unsupported or missing monitoringType",
				},
			},
		},
		{
			description: "TC4: Ipv4Addr-only target identification is not yet supported",
			afID:        "af1",
			monSub:      &monSub4ForAf1,
			expectedResponse: &HandlerResponse{
				Status: http.StatusBadRequest,
				Body: &models.ProblemDetails{
					Status: http.StatusBadRequest,
					Title:  "Malformed request syntax",
					Detail: "Ipv4Addr/Ipv6Addr-only target identification is not yet supported; provide ExternalId or Msisdn",
				},
			},
		},
		{
			description: "TC5: Missing one of MaximumNumberOfReports, MonitorExpireTime",
			afID:        "af1",
			monSub:      &monSub5ForAf1,
			expectedResponse: &HandlerResponse{
				Status: http.StatusBadRequest,
				Body: &models.ProblemDetails{
					Status: http.StatusBadRequest,
					Title:  "Malformed request syntax",
					Detail: "Missing one of MaximumNumberOfReports, MonitorExpireTime",
				},
			},
		},
	}

	nefCtx := nefApp.Context()
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			httpRecorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(httpRecorder)

			nefApp.Processor().PostMonitoringEventSubscription(c, tc.afID, tc.monSub)
			require.Equal(t, tc.expectedResponse.Status, httpRecorder.Code)

			if tc.expectedResponse.Headers != nil {
				for k, v := range tc.expectedResponse.Headers {
					resp := httpRecorder.Result()
					defer func() {
						if err := resp.Body.Close(); err != nil {
							t.Errorf("failed to close resp body: %v", err)
						}
					}()

					require.ElementsMatch(t, v, resp.Header.Values(k))
				}
			}

			assertJSONBodyEqual(t, tc.expectedResponse.Body, httpRecorder.Body.Bytes())
		})
	}
	nefCtx.DeleteAf("af1")
	nefCtx.ResetCorreID()
}

// TestPostMonitoringEventSubscription_AmfError verifies that an AMF-side failure is
// surfaced to the AF as the ProblemDetails AMF returned, rather than a generic error.
func TestPostMonitoringEventSubscription_AmfError(t *testing.T) {
	initNRFDiscUDMSdmStub()
	initUDMSdmGetSupiOrGpsiStub(http.StatusOK, "imsi-208930000000001")
	initNRFDiscAMFStub()
	initAMFEvtsCreateSubscriptionErrorStub(http.StatusForbidden)

	monSub := monSub1ForAf1

	httpRecorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(httpRecorder)

	nefApp.Processor().PostMonitoringEventSubscription(c, "af1", &monSub)
	require.Equal(t, http.StatusForbidden, httpRecorder.Code)

	nefApp.Context().DeleteAf("af1")
	nefApp.Context().ResetCorreID()
}

func TestDeleteIndividualMonitoringEventSubscription(t *testing.T) {
	initNRFDiscAMFStub()
	initAMFEvtsDeleteSubscriptionStub(http.StatusNoContent)

	testCases := []struct {
		description      string
		afID             string
		subID            string
		expectedResponse *HandlerResponse
	}{
		{
			description: "TC1: Successful delete monitoring event subscription",
			afID:        "af1",
			subID:       "1",
			expectedResponse: &HandlerResponse{
				Status: http.StatusNoContent,
			},
		},
		{
			description: "TC2: Delete non-existent monitoring event subscription",
			afID:        "af1",
			subID:       "2",
			expectedResponse: &HandlerResponse{
				Status: http.StatusNotFound,
				Body: &models.ProblemDetails{
					Status: http.StatusNotFound,
					Title:  "Data not found",
					Detail: "Subscription is not found",
				},
			},
		},
	}

	nefCtx := nefApp.Context()
	af1 := nefCtx.NewAf("af1")
	af1.Mu.Lock()
	correID1 := nefCtx.NewCorreID()
	monSubCtx1 := af1.NewMonSub(correID1, &monSub1ForAf1)
	monSubCtx1.AmfSubID = "amf-sub-1"
	af1.MonSubs[monSubCtx1.SubID] = monSubCtx1
	nefCtx.AddAf(af1)
	af1.Mu.Unlock()

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			httpRecorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(httpRecorder)

			nefApp.Processor().DeleteIndividualMonitoringEventSubscription(c, tc.afID, tc.subID)
			c.Writer.WriteHeaderNow()
			require.Equal(t, tc.expectedResponse.Status, httpRecorder.Code)

			assertJSONBodyEqual(t, tc.expectedResponse.Body, httpRecorder.Body.Bytes())
		})
	}
	nefCtx.DeleteAf(af1.AfID)
	nefCtx.ResetCorreID()
}

func initNRFDiscAMFStub() {
	searchResult := &models.Nrf_NFDisc_SearchResult{
		ValidityPeriod: 100,
		NfInstances: []models.Nrf_NFDisc_NFProfile{
			{
				NfInstanceId: "nef-unit-testing",
				NfType:       "AMF",
				NfStatus:     "REGISTERED",
				Ipv4Addresses: []string{
					"127.0.0.18",
				},
				NfServices: []models.Nrf_NFDisc_NFService{
					{
						ServiceInstanceId: "1",
						ServiceName:       "namf-evts",
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
								Ipv4Address: "127.0.0.18",
								Transport:   "TCP",
								Port:        8000,
							},
						},
						ApiPrefix: "http://127.0.0.18:8000",
					},
				},
			},
		},
	}

	gock.New("http://127.0.0.10:8000/nnrf-disc/v1").
		Get("/nf-instances").
		MatchParam("target-nf-type", "AMF").
		MatchParam("requester-nf-type", "NEF").
		MatchParam("service-names", "namf-evts").
		Reply(http.StatusOK).
		JSON(searchResult)
}

// initAMFEvtsCreateSubscriptionStub matches exactly one POST, matching the single
// successful create in TestPostMonitoringEventSubscription's TC1; gock auto-disables it
// after that match, so it can't bleed into any other test.
func initAMFEvtsCreateSubscriptionStub(statusCode int, amfSubID string) {
	rsp := models.Amf_EvtExpos_AmfCreatedEventSubscription{
		SubscriptionId: amfSubID,
	}
	gock.New("http://127.0.0.18:8000/namf-evts/v1").
		Post("/subscriptions").
		Reply(statusCode).
		JSON(rsp)
}

func initAMFEvtsCreateSubscriptionErrorStub(statusCode int) {
	pd := models.ProblemDetails{
		Status: int32(statusCode),
		Title:  "Forbidden",
		Detail: "subscription rejected by AMF",
	}
	gock.New("http://127.0.0.18:8000/namf-evts/v1").
		Post("/subscriptions").
		Reply(statusCode).
		JSON(pd)
}

func initAMFEvtsDeleteSubscriptionStub(statusCode int) {
	gock.New("http://127.0.0.18:8000/namf-evts/v1").
		Delete("/subscriptions/.*").
		Reply(statusCode)
}

func initNRFDiscUDMSdmStub() {
	searchResult := &models.Nrf_NFDisc_SearchResult{
		ValidityPeriod: 100,
		NfInstances: []models.Nrf_NFDisc_NFProfile{
			{
				NfInstanceId: "nef-unit-testing",
				NfType:       "UDM",
				NfStatus:     "REGISTERED",
				Ipv4Addresses: []string{
					"127.0.0.3",
				},
				NfServices: []models.Nrf_NFDisc_NFService{
					{
						ServiceInstanceId: "1",
						ServiceName:       "nudm-sdm",
						Versions: []models.Nrf_NFMgmt_NFServiceVersion{
							{
								ApiVersionInUri: "v2",
								ApiFullVersion:  "1.0.0",
							},
						},
						Scheme:          "http",
						NfServiceStatus: "REGISTERED",
						IpEndPoints: []models.Nrf_NFMgmt_IpEndPoint{
							{
								Ipv4Address: "127.0.0.3",
								Transport:   "TCP",
								Port:        8000,
							},
						},
						ApiPrefix: "http://127.0.0.3:8000",
					},
				},
			},
		},
	}

	gock.New("http://127.0.0.10:8000/nnrf-disc/v1").
		Get("/nf-instances").
		MatchParam("target-nf-type", "UDM").
		MatchParam("requester-nf-type", "NEF").
		MatchParam("service-names", "nudm-sdm").
		Reply(http.StatusOK).
		JSON(searchResult)
}

func initUDMSdmGetSupiOrGpsiStub(statusCode int, supi string) {
	rsp := models.Udm_SDM_IdTranslationResult{
		Supi: supi,
	}
	gock.New("http://127.0.0.3:8000/nudm-sdm/v2").
		Get("/.*/id-translation-result").
		Reply(statusCode).
		JSON(rsp)
}
