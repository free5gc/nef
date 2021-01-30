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
