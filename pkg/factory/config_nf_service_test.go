package factory

import (
	"testing"

	"github.com/free5gc/openapi/models"
	"github.com/free5gc/util/serviceuri"
)

// The registered NFService must follow TS 29.510 (no URL in apiPrefix, a
// hostname as fqdn, an IP literal in ipEndPoints) and resolve back to the NEF
// API root.
func TestNFServicesAddressing(t *testing.T) {
	tests := []struct {
		registerAddr string
		wantFQDN     string
		wantIPv4     string
		wantAPIRoot  string
	}{
		{"10.0.0.5", "", "10.0.0.5", "http://10.0.0.5:8000"},
		{"nef.free5gc.org", "nef.free5gc.org", "", "http://nef.free5gc.org:8000"},
	}

	for _, tt := range tests {
		t.Run(tt.registerAddr, func(t *testing.T) {
			c := &Config{
				Info: &Info{Version: "1.0.0"},
				Configuration: &Configuration{
					Sbi:         &Sbi{Scheme: "http", RegisterIPv4: tt.registerAddr, Port: 8000},
					ServiceList: []Service{{ServiceName: string(models.Nrf_NFMgmt_ServiceName_NNEF_PFDMANAGEMENT)}},
				},
			}
			s := c.NFServices()[0]

			if s.ApiPrefix != "" || s.Fqdn != tt.wantFQDN ||
				len(s.IpEndPoints) != 1 || s.IpEndPoints[0].Ipv4Address != tt.wantIPv4 {
				t.Fatalf("unexpected NFService addressing: %+v", s)
			}

			profile := &models.Nrf_NFDisc_NFProfile{NfServices: []models.Nrf_NFDisc_NFService{{
				ServiceName:     s.ServiceName,
				NfServiceStatus: s.NfServiceStatus,
				Scheme:          s.Scheme,
				Fqdn:            s.Fqdn,
				IpEndPoints:     s.IpEndPoints,
			}}}
			got, err := serviceuri.ResolveNFServiceURI(profile, s.ServiceName, serviceuri.Options{})
			if err != nil || got.APIRoot != tt.wantAPIRoot {
				t.Fatalf("resolved %q, error %v; want %q", got.APIRoot, err, tt.wantAPIRoot)
			}
		})
	}
}
