package util

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/free5gc/openapi/models"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

const (
	validToken   = "valid"
	invalidToken = "invalid"
)

type mockNEFContext struct{}

func newMockNEFContext() *mockNEFContext {
	return &mockNEFContext{}
}

func (m *mockNEFContext) AuthorizationCheck(token string, serviceName models.ServiceName) error {
	if token == validToken {
		return nil
	}

	return errors.New("invalid token")
}

func TestRouterAuthorizationCheckCheck(t *testing.T) {
	tests := []struct {
		name       string
		token      string
		statusCode int
	}{
		{
			name:       "valid token",
			token:      validToken,
			statusCode: http.StatusOK,
		},
		{
			name:       "invalid token",
			token:      invalidToken,
			statusCode: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			if err != nil {
				t.Fatalf("create request: %v", err)
			}
			req.Header.Set("Authorization", tc.token)
			c.Request = req

			rac := NewRouterAuthorizationCheck(models.ServiceName("testService"))
			rac.Check(c, newMockNEFContext())

			if w.Code != tc.statusCode {
				t.Fatalf("status code = %d, want %d", w.Code, tc.statusCode)
			}
		})
	}
}
