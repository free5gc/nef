package processor

import (
	"net/http"
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
