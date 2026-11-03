package sbi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	nfcontext "github.com/free5gc/nef/internal/context"
	"github.com/free5gc/nef/pkg/factory"
	"github.com/stretchr/testify/require"
)

type callbackOAuthApp struct {
	nef
	cfg *factory.Config
	ctx *nfcontext.NefContext
}

func (a *callbackOAuthApp) Config() *factory.Config        { return a.cfg }
func (a *callbackOAuthApp) Context() *nfcontext.NefContext { return a.ctx }

func TestCallbackOAuthRoute(t *testing.T) {
	app := &callbackOAuthApp{cfg: &factory.Config{Configuration: &factory.Configuration{
		Sbi:         &factory.Sbi{BindingIPv4: "127.0.0.1", Port: 0},
		ServiceList: []factory.Service{{ServiceName: factory.ServiceTraffInflu}},
	}}}
	ctx, err := nfcontext.NewContext(app)
	require.NoError(t, err)
	app.ctx = ctx
	app.ctx.OAuth2Required = true
	server, err := NewServer(app, "")
	require.NoError(t, err)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
		factory.NefCallbackResUriPrefix+"/notification/smf", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.router.ServeHTTP(response, req)
	require.Equal(t, http.StatusBadRequest, response.Code)

	req = httptest.NewRequestWithContext(t.Context(), http.MethodPost,
		factory.TraffInfluResUriPrefix+"/af/subscriptions", strings.NewReader("{"))
	response = httptest.NewRecorder()
	server.router.ServeHTTP(response, req)
	require.Equal(t, http.StatusUnauthorized, response.Code)
}
