package consumer

import (
	"testing"

	nef_context "github.com/free5gc/nef/internal/context"
	"github.com/free5gc/nef/pkg/app"
	"github.com/free5gc/nef/pkg/factory"
)

// nefTestApp satisfies the nef interface the consumer needs. It embeds app.App
// so an unexpected call panics instead of silently returning a zero value, the
// same shape the processor tests use.
type nefTestApp struct {
	app.App

	cfg    *factory.Config
	nefCtx *nef_context.NefContext
}

func (a *nefTestApp) Config() *factory.Config          { return a.cfg }
func (a *nefTestApp) Context() *nef_context.NefContext { return a.nefCtx }

// newTestConfig is the smallest config buildNfProfile accepts: it needs at
// least one service to advertise.
func newTestConfig() *factory.Config {
	return &factory.Config{
		Info: &factory.Info{Version: "1.0.0", Description: "NEF consumer test configuration"},
		Configuration: &factory.Configuration{
			NfInstanceId: testNfId,
			NrfUri:       testNrfUri,
			Sbi: &factory.Sbi{
				Scheme:       "http",
				RegisterIPv4: "127.0.0.10",
				BindingIPv4:  "127.0.0.10",
				Port:         8000,
			},
			ServiceList: []factory.Service{{ServiceName: "3gpp-traffic-influence"}},
		},
	}
}

func newTestConsumer(t *testing.T, cfg *factory.Config) *Consumer {
	t.Helper()

	testApp := &nefTestApp{cfg: cfg}

	nefCtx, err := nef_context.NewContext(testApp)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	testApp.nefCtx = nefCtx

	testConsumer, err := NewConsumer(testApp)
	if err != nil {
		t.Fatalf("NewConsumer: %v", err)
	}

	return testConsumer
}
