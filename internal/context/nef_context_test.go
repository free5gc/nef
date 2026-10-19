package context

import (
	tlsContext "context"
	tlsX509 "crypto/x509"
	tlsErrors "errors"
	tlsIO "io"
	tlsLog "log"
	tlsHTTP "net/http"
	tlsServer "net/http/httptest"
	"sync"
	tlsAtomic "sync/atomic"
	"testing"

	tlsFactory "github.com/free5gc/nef/pkg/factory"
	tlsOpenAPI "github.com/free5gc/openapi"
	tlsNFM "github.com/free5gc/openapi/nrf/NFMgmt"
	tlsYAML "gopkg.in/yaml.v2"
)

func TestGetOrCreateAfConcurrent(t *testing.T) {
	nefCtx := &NefContext{afs: make(map[string]*AfData)}

	const goroutineCount = 32
	start := make(chan struct{})
	results := make(chan *AfData, goroutineCount)
	var wg sync.WaitGroup
	for range goroutineCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results <- nefCtx.GetOrCreateAf("af1")
		}()
	}

	close(start)
	wg.Wait()
	close(results)

	var first *AfData
	for af := range results {
		if first == nil {
			first = af
			continue
		}
		if af != first {
			t.Fatal("GetOrCreateAf returned different AF instances for the same ID")
		}
	}

	if first == nil {
		t.Fatal("GetOrCreateAf returned no AF instance")
	}
	if got := nefCtx.GetAf("af1"); got != first {
		t.Fatal("GetOrCreateAf did not register the returned AF instance")
	}
}

// Exercise parsed NF config through the same startup configuration API and
// generated NRF client used in production. No deployment files are modified.
func TestSBIConfigTLSVerification(t *testing.T) {
	t.Cleanup(func() {
		_ = tlsOpenAPI.ConfigureHTTP2Clients(tlsOpenAPI.TransportReadIdleTimeout, tlsOpenAPI.TransportPingTimeout, false)
	})
	var requests tlsAtomic.Int32
	server := tlsServer.NewUnstartedServer(tlsHTTP.HandlerFunc(func(w tlsHTTP.ResponseWriter, r *tlsHTTP.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = tlsIO.WriteString(w, `{}`)
	}))
	server.EnableHTTP2 = true
	server.Config.ErrorLog = tlsLog.New(tlsIO.Discard, "", 0)
	server.StartTLS()
	defer server.Close()
	for _, value := range []string{"", "false", "true"} {
		t.Run("verify="+value, func(t *testing.T) {
			data := "configuration:\n  sbi:\n    scheme: https\n"
			if value != "" {
				data += "    tls:\n      verify: " + value + "\n"
			}
			var cfg tlsFactory.Config
			if err := tlsYAML.Unmarshal([]byte(data), &cfg); err != nil {
				t.Fatal(err)
			}
			if cfg.GetTLSVerify() != (value == "true") {
				t.Fatal("verify parsing mismatch")
			}
			if err := tlsOpenAPI.ConfigureHTTP2Clients(cfg.GetReadIdleTimeout(), cfg.GetPingTimeout(), cfg.GetTLSVerify()); err != nil {
				t.Fatal(err)
			}
			apiCfg := tlsNFM.NewConfiguration()
			apiCfg.SetBasePath(server.URL)
			client := tlsNFM.NewAPIClient(apiCfg)
			req := &tlsNFM.GetNFInstanceRequest{}
			req.SetNfInstanceID("tls-test")
			before := requests.Load()
			_, err := client.NFInstanceIDDocumentApi.GetNFInstance(tlsContext.Background(), req)
			if value == "true" {
				var unknown tlsX509.UnknownAuthorityError
				if !tlsErrors.As(err, &unknown) {
					t.Fatalf("expected unknown authority, got %v", err)
				}
				if requests.Load() != before {
					t.Fatal("HTTP handler reached despite rejected certificate")
				}
			} else if err != nil {
				t.Fatal(err)
			} else if requests.Load() != before+1 {
				t.Fatal("request not received")
			}
		})
	}
}
