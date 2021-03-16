package processor

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/h2non/gock.v1"

	"bitbucket.org/free5gc-team/openapi/models"
)

func TestGetApplicationsPFD(t *testing.T) {
	initUDRDrGetPfdDatasStub()
	defer gock.Off()

	testCases := []struct {
		description      string
		appIDs           []string
		expectedResponse *HandlerResponse
	}{
		{
			description: "All App IDs found, should return all PfdDataforApp",
			appIDs:      []string{"app1", "app2"},
			expectedResponse: &HandlerResponse{
				Status: http.StatusOK,
				Body:   &[]models.PfdDataForApp{pfdDataForApp1, pfdDataForApp2},
			},
		},
		{
			description: "All App ID not found, should return ProblemDetails",
			appIDs:      []string{"app3"},
			expectedResponse: &HandlerResponse{
				Status: http.StatusNotFound,
				Body:   &models.ProblemDetails{Status: http.StatusNotFound},
			},
		},
	}

	Convey("Given App IDs, should get a list of PfdDataForApp", t, func() {
		for _, tc := range testCases {
			Convey(tc.description, func() {
				rsp := nefProcessor.GetApplicationsPFD(tc.appIDs)
				So(rsp, ShouldResemble, tc.expectedResponse)
			})
		}
	})
}

func TestGetIndividualApplicationPFD(t *testing.T) {
	initUDRDrGetPfdDataStub()
	defer gock.Off()

	testCases := []struct {
		description      string
		appID            string
		expectedResponse *HandlerResponse
	}{
		{
			description: "App ID found, should return the PfdDataforApp",
			appID:       "app1",
			expectedResponse: &HandlerResponse{
				Status: http.StatusOK,
				Body:   &pfdDataForApp1,
			},
		},
		{
			description: "App ID not found, should return ProblemDetails",
			appID:       "app3",
			expectedResponse: &HandlerResponse{
				Status: http.StatusNotFound,
				Body:   &models.ProblemDetails{Status: http.StatusNotFound},
			},
		},
	}

	Convey("Given App IDs, should get a list of PfdDataForApp", t, func() {
		for _, tc := range testCases {
			Convey(tc.description, func() {
				rsp := nefProcessor.GetIndividualApplicationPFD(tc.appID)
				So(rsp, ShouldResemble, tc.expectedResponse)
			})
		}
	})
}

func TestPostPFDSubscriptions(t *testing.T) {
	pfdSubsc := &models.PfdSubscription{
		ApplicationIds: []string{"app1", "app2"},
		NotifyUri:      "http://pfdSub1URI/notify",
	}

	testCases := []struct {
		description      string
		subscription     *models.PfdSubscription
		expectedResponse *HandlerResponse
	}{
		{
			description:  "Successful subscription, should return PfdSubscription",
			subscription: pfdSubsc,
			expectedResponse: &HandlerResponse{
				Status:  http.StatusCreated,
				Headers: map[string][]string{"Location": {genPfdSubscriptionURI(nefProcessor.cfg.GetSbiUri(), "1")}},
				Body:    pfdSubsc,
			},
		},
	}

	Convey("Given a subscription, should store it and return the resource URI", t, func() {
		for _, tc := range testCases {
			Convey(tc.description, func() {
				rsp := nefProcessor.PostPFDSubscriptions(tc.subscription)
				So(rsp, ShouldResemble, tc.expectedResponse)
			})
		}
	})
}

func TestDeleteIndividualPFDSubscription(t *testing.T) {
	testCases := []struct {
		description      string
		subscriptionID   string
		expectedResponse *HandlerResponse
	}{
		{
			description:    "Successful unsubscription",
			subscriptionID: "1",
			expectedResponse: &HandlerResponse{
				Status: http.StatusNoContent,
			},
		},
	}

	Convey("Given a subscription ID, should delete the specified subscription", t, func() {
		for _, tc := range testCases {
			Convey(tc.description, func() {
				rsp := nefProcessor.DeleteIndividualPFDSubscription(tc.subscriptionID)
				So(rsp, ShouldResemble, tc.expectedResponse)
			})
		}
	})
}

var (
	// `notifChan` are used in `TestPostPfdChangeReports()` to pass the notification requests intercepted by gock.
	notifChan   = make(chan *http.Request)
	pfdContent1 = models.PfdContent{
		PfdId: "pfd1",
		FlowDescriptions: []string{
			"permit in ip from 10.68.28.39 80 to any",
			"permit out ip from any to 10.68.28.39 80",
		},
	}
)

func TestPostPfdChangeReports(t *testing.T) {
	// Note: Because TestPostPFDSubscriptions() already used subscription ID 1, the ID will start from 2 here.
	initUDRDrPutPfdDataStub(http.StatusOK)
	initUDRDrDeletePfdDataStub()
	initNEFNotificationStub("http://pfdSub2URI")
	initNEFNotificationStub("http://pfdSub3URI")
	defer gock.Off()
	gock.Observe(func(request *http.Request, mock gock.Mock) {
		if strings.Contains(request.URL.String(), "pfdSub") {
			notifChan <- request
		}
	})

	afCtx := nefContext.NewAfCtx("af1")
	nefContext.AddAfCtx(afCtx)
	defer nefContext.DeleteAfCtx("af1")
	afPfdTans := nefContext.NewAfPfdTrans(afCtx)
	afCtx.AddPfdTrans(afPfdTans)
	afPfdTans.AddExtAppID("app1")
	afPfdTans.AddExtAppID("app2")

	nefProcessor.notifier.PfdChangeNotifier.AddPfdSub(&models.PfdSubscription{
		ApplicationIds: []string{"app1"},
		NotifyUri:      "http://pfdSub2URI",
	})
	nefProcessor.notifier.PfdChangeNotifier.AddPfdSub(&models.PfdSubscription{
		ApplicationIds: []string{"app1", "app2"},
		NotifyUri:      "http://pfdSub3URI",
	})
	defer func() {
		if err := nefProcessor.notifier.PfdChangeNotifier.DeletePfdSub("2"); err != nil {
			t.Fatal(err)
		}
		if err := nefProcessor.notifier.PfdChangeNotifier.DeletePfdSub("3"); err != nil {
			t.Fatal(err)
		}
	}()

	testCases := []struct {
		description           string
		triggerFunc           func()
		expectedNotifications map[string][]models.PfdChangeNotification
	}{
		{
			description: "Update app1, should send notification for subscription 2 and 3",
			triggerFunc: func() {
				nefProcessor.PutIndividualApplicationPFDManagement("af1", "1", "app1", &models.PfdData{
					ExternalAppId: "app1",
					Pfds: map[string]models.Pfd{
						"pfd1": pfd1,
					},
				})
			},
			expectedNotifications: map[string][]models.PfdChangeNotification{
				"http://pfdSub2URI/notify": {
					{
						ApplicationId: "app1",
						Pfds: []models.PfdContent{
							pfdContent1,
						},
					},
				},
				"http://pfdSub3URI/notify": {
					{
						ApplicationId: "app1",
						Pfds: []models.PfdContent{
							pfdContent1,
						},
					},
				},
			},
		},
		{
			description: "Delete app2, should send notification for subscription 3",
			triggerFunc: func() {
				nefProcessor.DeleteIndividualApplicationPFDManagement("af1", "1", "app2")
			},
			expectedNotifications: map[string][]models.PfdChangeNotification{
				"http://pfdSub3URI/notify": {
					{
						ApplicationId: "app2",
						RemovalFlag:   true,
					},
				},
			},
		},
	}

	Convey("Subscribe for appIds, should receive notifications when their Pfds changing", t, func() {
		for _, tc := range testCases {
			Convey(tc.description, func() {
				tc.triggerFunc()
				for i := 0; i < len(tc.expectedNotifications); i++ {
					r := <-notifChan

					var getNotifications []models.PfdChangeNotification
					if err := json.NewDecoder(r.Body).Decode(&getNotifications); err != nil {
						t.Fatal(err)
					}
					So(tc.expectedNotifications, ShouldContainKey, r.URL.String())
					So(tc.expectedNotifications[r.URL.String()], ShouldResemble, getNotifications)
				}
			})
		}
	})
}

func initNEFNotificationStub(notifyURI string) {
	gock.New(notifyURI).
		Post("/notify").
		Persist().
		Reply(http.StatusNoContent)
}
