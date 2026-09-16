package processor

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/free5gc/openapi/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
)

func TestTranslateUeId(t *testing.T) {
	testCases := []struct {
		description      string
		afID             string
		req              models.Nef_UEId_UeIdTranslationReqData
		setupMocks       func()
		expectedResponse *HandlerResponse
	}{
		{
			description: "TC1: Valid GPSI, should return SUPI",
			afID:        "af1",
			req:         models.Nef_UEId_UeIdTranslationReqData{Gpsi: "msisdn-0900000001"},
			setupMocks: func() {
				initNRFDiscUDMSdmStub()
				initUDMSdmGetSupiOrGpsiStub(http.StatusOK, "imsi-208930000000001")
			},
			expectedResponse: &HandlerResponse{
				Status: http.StatusOK,
				Body: &models.Nef_UEId_UeIdTranslationRspData{
					Supi: "imsi-208930000000001",
					Gpsi: "msisdn-0900000001",
				},
			},
		},
		{
			description: "TC2: Missing GPSI, should return 400",
			afID:        "af1",
			req:         models.Nef_UEId_UeIdTranslationReqData{},
			setupMocks:  func() {},
			expectedResponse: &HandlerResponse{
				Status: http.StatusBadRequest,
				Body: &models.ProblemDetails{
					Status: http.StatusBadRequest,
					Title:  "Malformed request syntax",
					Detail: "gpsi is required",
				},
			},
		},
		{
			description: "TC3: AF not found, should return 404",
			afID:        "unknown-af",
			req:         models.Nef_UEId_UeIdTranslationReqData{Gpsi: "msisdn-0900000001"},
			setupMocks:  func() {},
			expectedResponse: &HandlerResponse{
				Status: http.StatusNotFound,
				Body: &models.ProblemDetails{
					Status: http.StatusNotFound,
					Title:  "Data not found",
					Detail: "AF not found: unknown-af",
				},
			},
		},
		{
			description: "TC4: GPSI not found in UDM, should return 404",
			afID:        "af1",
			req:         models.Nef_UEId_UeIdTranslationReqData{Gpsi: "msisdn-0000000000"},
			setupMocks: func() {
				gock.New("http://127.0.0.3:8000/nudm-sdm/v2").
					Get("/.*/id-translation-result").
					Reply(http.StatusNotFound).
					JSON(models.ProblemDetails{Status: http.StatusNotFound, Title: "User not found"})
			},
			expectedResponse: &HandlerResponse{
				Status: http.StatusNotFound,
				Body:   &models.ProblemDetails{Status: http.StatusNotFound, Title: "User not found"},
			},
		},
	}

	nefCtx := nefApp.Context()
	af1 := nefCtx.NewAf("af1")
	nefCtx.AddAf(af1)
	defer nefCtx.DeleteAf("af1")

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			tc.setupMocks()
			defer gock.Off()

			httpRecorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(httpRecorder)

			nefApp.Processor().TranslateUeId(c, tc.afID, &tc.req)
			require.Equal(t, tc.expectedResponse.Status, httpRecorder.Code)

			assertJSONBodyEqual(t, tc.expectedResponse.Body, httpRecorder.Body.Bytes())
		})
	}
}
