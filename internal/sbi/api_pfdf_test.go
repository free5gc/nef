package sbi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestParseSupportedFeatures(t *testing.T) {
	testCases := []struct {
		description      string
		query            string
		expectedValue    *string
		expectedHTTPCode int
	}{
		{
			description:      "TC1: no supported-features query",
			query:            "",
			expectedValue:    nil,
			expectedHTTPCode: 0,
		},
		{
			description:      "TC2: single supported-features query",
			query:            "?supported-features=1",
			expectedValue:    ptrString("1"),
			expectedHTTPCode: 0,
		},
		{
			description:      "TC3: duplicated supported-features query",
			query:            "?supported-features=1&supported-features=2",
			expectedValue:    nil,
			expectedHTTPCode: http.StatusBadRequest,
		},
		{
			description:      "TC4: empty supported-features query",
			query:            "?supported-features=",
			expectedValue:    nil,
			expectedHTTPCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			httpRecorder := httptest.NewRecorder()
			gc, _ := gin.CreateTestContext(httpRecorder)
			gc.Request = httptest.NewRequest(http.MethodGet, "/nnef-pfdmanagement/v1/applications/1"+tc.query, nil)

			value, pd := parseSupportedFeatures(gc)
			require.Equal(t, tc.expectedValue, value)
			if tc.expectedHTTPCode == 0 {
				require.Nil(t, pd)
			} else {
				require.NotNil(t, pd)
				require.Equal(t, int(pd.Status), tc.expectedHTTPCode)
			}
		})
	}
}

func ptrString(v string) *string {
	return &v
}
