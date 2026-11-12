package processor

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/gin-gonic/gin"
	"github.com/h2non/gock"
	"github.com/stretchr/testify/require"
)

func TestGetApplicationsPFD(t *testing.T) {
	openapi.InterceptInnerHttp2Client(t, false)
	initNRFDiscUDRStub()
	initUDRDrGetPfdDatasStub()
	defer gock.Off()

	testCases := []struct {
		description      string
		appIDs           []string
		expectedResponse *HandlerResponse
	}{
		{
			description: "TC1: All App IDs found, should return all PfdDataforApp",
			appIDs:      []string{"app1", "app2"},
			expectedResponse: &HandlerResponse{
				Status: http.StatusOK,
				Body:   &[]models.Nef_PFDMgmt_PfdDataForApp{pfdDataForApp1, pfdDataForApp2},
			},
		},
		{
			description: "TC2: All App ID not found, should return ProblemDetails",
			appIDs:      []string{"app3"},
			expectedResponse: &HandlerResponse{
				Status: http.StatusNotFound,
				Body:   &models.ProblemDetails{Status: http.StatusNotFound},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			httpRecorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(httpRecorder)

			nefApp.Processor().GetApplicationsPFD(c, tc.appIDs)
			require.Equal(t, tc.expectedResponse.Status, httpRecorder.Code)

			assertJSONBodyEqual(t, tc.expectedResponse.Body, httpRecorder.Body.Bytes())
		})
	}
}

func TestGetApplicationsPFDWithMalformedMultipartFromUDR(t *testing.T) {
	openapi.InterceptInnerHttp2Client(t, false)
	initNRFDiscUDRStub()
	initUDRDrGetPfdDatasMultipartStub()
	defer gock.Off()

	httpRecorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(httpRecorder)

	require.NotPanics(t, func() {
		nefApp.Processor().GetApplicationsPFD(c, []string{"app1"})
	})
	require.Equal(t, http.StatusInternalServerError, httpRecorder.Code)
}

func TestGetIndividualApplicationPFD(t *testing.T) {
	openapi.InterceptInnerHttp2Client(t, false)
	initNRFDiscUDRStub()
	initUDRDrGetPfdDataStub()
	defer gock.Off()

	testCases := []struct {
		description      string
		appID            string
		expectedResponse *HandlerResponse
	}{
		{
			description: "TC1: App ID found, should return the PfdDataforApp",
			appID:       "app1",
			expectedResponse: &HandlerResponse{
				Status: http.StatusOK,
				Body:   &pfdDataForApp1,
			},
		},
		{
			description: "TC2: App ID not found, should return ProblemDetails",
			appID:       "app3",
			expectedResponse: &HandlerResponse{
				Status: http.StatusNotFound,
				Body:   &models.ProblemDetails{Status: http.StatusNotFound},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			httpRecorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(httpRecorder)

			nefApp.Processor().GetIndividualApplicationPFD(c, tc.appID)
			require.Equal(t, tc.expectedResponse.Status, httpRecorder.Code)

			assertJSONBodyEqual(t, tc.expectedResponse.Body, httpRecorder.Body.Bytes())
		})
	}
}

func TestPostPFDSubscriptions(t *testing.T) {
	openapi.InterceptInnerHttp2Client(t, false)
	pfdSubsc := &models.Nef_PFDMgmt_PfdSubscription{
		ApplicationIds: []string{"app1", "app2"},
		NotifyUri:      "http://pfdSub1URI/notify",
	}

	testCases := []struct {
		description      string
		subscription     *models.Nef_PFDMgmt_PfdSubscription
		expectedResponse *HandlerResponse
	}{
		{
			description:  "TC1: Successful subscription, should return PfdSubscription",
			subscription: pfdSubsc,
			expectedResponse: &HandlerResponse{
				Status: http.StatusCreated,
				Headers: map[string][]string{
					"Location": {nefApp.Processor().genPfdSubscriptionURI("1")},
				},
				Body: pfdSubsc,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			httpRecorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(httpRecorder)

			nefApp.Processor().PostPFDSubscriptions(c, tc.subscription)
			require.Equal(t, tc.expectedResponse.Status, httpRecorder.Code)

			assertJSONBodyEqual(t, tc.expectedResponse.Body, httpRecorder.Body.Bytes())
		})
	}
}

func TestDeleteIndividualPFDSubscription(t *testing.T) {
	openapi.InterceptInnerHttp2Client(t, false)
	testCases := []struct {
		description      string
		subscriptionID   string
		expectedResponse *HandlerResponse
	}{
		{
			description:    "TC1: Successful unsubscription",
			subscriptionID: "1",
			expectedResponse: &HandlerResponse{
				Status: http.StatusNoContent,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			httpRecorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(httpRecorder)

			nefApp.Processor().DeleteIndividualPFDSubscription(c, tc.subscriptionID)
			c.Writer.WriteHeaderNow()
			require.Equal(t, tc.expectedResponse.Status, httpRecorder.Code)

			assertJSONBodyEqual(t, tc.expectedResponse.Body, httpRecorder.Body.Bytes())
		})
	}
}

var pfdContent1 = models.Nef_PFDMgmt_PfdContent{
	PfdId: "pfd1",
	FlowDescriptions: []string{
		"permit in ip from 10.68.28.39 80 to any",
		"permit out ip from any to 10.68.28.39 80",
	},
}

func TestPostPfdChangeReports(t *testing.T) {
	openapi.InterceptInnerHttp2Client(t, false)
	initNRFDiscUDRStub()
	// Note: Because TestPostPFDSubscriptions() already used subscription ID 1, the ID will start from 2 here.
	initUDRDrPutPfdDataStub(http.StatusOK)
	initUDRDrDeletePfdDataStub()
	defer gock.Off()

	type notification struct {
		uri   string
		items []models.Nef_PFDMgmt_PfdChangeNotification
	}
	notifChan := make(chan notification, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("unexpected notification method: %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var items []models.Nef_PFDMgmt_PfdChangeNotification
		if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
			t.Errorf("decode PFD notification: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		select {
		case notifChan <- notification{uri: "http://" + r.Host + r.URL.Path, items: items}:
		default:
			t.Error("unexpected extra PFD notification")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	af := nefApp.Context().NewAf("af1")
	nefApp.Context().AddAf(af)
	defer nefApp.Context().DeleteAf("af1")

	af.Mu.Lock()
	afPfdTr := af.NewPfdTrans()
	af.PfdTrans[afPfdTr.TransID] = afPfdTr
	afPfdTr.AddExtAppID("app1")
	afPfdTr.AddExtAppID("app2")
	af.Mu.Unlock()

	subsID1 := nefApp.Notifier().PfdChangeNotifier.AddPfdSub(&models.Nef_PFDMgmt_PfdSubscription{
		ApplicationIds: []string{"app1"},
		NotifyUri:      server.URL + "/sub2/notify",
	})
	subsID2 := nefApp.Notifier().PfdChangeNotifier.AddPfdSub(&models.Nef_PFDMgmt_PfdSubscription{
		ApplicationIds: []string{"app1", "app2"},
		NotifyUri:      server.URL + "/sub3/notify",
	})
	defer func() {
		if err := nefApp.Notifier().PfdChangeNotifier.DeletePfdSub(subsID1); err != nil {
			t.Fatal(err)
		}
		if err := nefApp.Notifier().PfdChangeNotifier.DeletePfdSub(subsID2); err != nil {
			t.Fatal(err)
		}
	}()

	testCases := []struct {
		description           string
		triggerFunc           func(c *gin.Context)
		expectedNotifications map[string][]models.Nef_PFDMgmt_PfdChangeNotification
	}{
		{
			description: "Update app1, should send notification for subscription 2 and 3",
			triggerFunc: func(c *gin.Context) {
				nefApp.Processor().PutIndividualApplicationPFDManagement(c, "af1", "1", "app1", &models.Nef_T8PFDMgmt_PfdData{
					ExternalAppId: "app1",
					Pfds: map[string]models.Nef_T8PFDMgmt_Pfd{
						"pfd1": pfd1,
					},
				})
			},
			expectedNotifications: map[string][]models.Nef_PFDMgmt_PfdChangeNotification{
				server.URL + "/sub2/notify": {
					{
						ApplicationId: "app1",
						Pfds: []models.Nef_PFDMgmt_PfdContent{
							pfdContent1,
						},
					},
				},
				server.URL + "/sub3/notify": {
					{
						ApplicationId: "app1",
						Pfds: []models.Nef_PFDMgmt_PfdContent{
							pfdContent1,
						},
					},
				},
			},
		},
		{
			description: "Delete app2, should send notification for subscription 3",
			triggerFunc: func(c *gin.Context) {
				nefApp.Processor().DeleteIndividualApplicationPFDManagement(c, "af1", "1", "app2")
			},
			expectedNotifications: map[string][]models.Nef_PFDMgmt_PfdChangeNotification{
				server.URL + "/sub3/notify": {
					{
						ApplicationId: "app2",
						RemovalFlag:   true,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			httpRecorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(httpRecorder)

			tc.triggerFunc(c)
			for i := 0; i < len(tc.expectedNotifications); i++ {
				select {
				case received := <-notifChan:
					require.Equal(t, tc.expectedNotifications[received.uri], received.items)
				case <-time.After(5 * time.Second):
					t.Fatal("timed out waiting for PFD notification")
				}
			}
		})
	}
}
