package consumer

import (
	"github.com/free5gc/openapi/models"
	"testing"
)

func TestNFServiceURIResolver(t *testing.T) {
	cases := []struct {
		name             string
		scheme           models.UriScheme
		fqdn, prefix, ip string
		port             int32
		want             string
	}{
		{"HTTP_IP", models.UriScheme_HTTP, "", "", "10.101.200.18", 8000, "http://10.101.200.18:8000"},
		{"HTTP_Path", models.UriScheme_HTTP, "", "/operator/edge", "10.101.200.18", 8000, "http://10.101.200.18:8000/operator/edge"},
		{"HTTPS_FQDN", models.UriScheme_HTTPS, "amf.example.org", "/sbi", "", 8443, "https://amf.example.org:8443/sbi"},
		{"Legacy", models.UriScheme_HTTP, "", "http://10.101.200.18:8000", "", 0, "http://10.101.200.18:8000"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			svc := models.Nrf_NFDisc_NFService{
				ServiceName:     models.Nrf_NFMgmt_ServiceName_NAMF_COMM,
				NfServiceStatus: models.Nrf_NFMgmt_NFServiceStatus_REGISTERED,
				Scheme:          tt.scheme, Fqdn: tt.fqdn, ApiPrefix: tt.prefix,
			}
			if tt.ip != "" || tt.port != 0 {
				svc.IpEndPoints = []models.Nrf_NFMgmt_IpEndPoint{{Ipv4Address: tt.ip, Port: tt.port}}
			}
			got, err := resolveNFServiceURI(models.Nrf_NFDisc_NFProfile{NfServices: []models.Nrf_NFDisc_NFService{svc}}, models.Nrf_NFMgmt_ServiceName_NAMF_COMM, models.Nrf_NFMgmt_NFServiceStatus_REGISTERED)
			if err != nil || got != tt.want {
				t.Fatalf("got %q err %v want %q", got, err, tt.want)
			}
		})
	}
	t.Run("RejectBadScheme", func(t *testing.T) {
		svc := models.Nrf_NFDisc_NFService{ServiceName: models.Nrf_NFMgmt_ServiceName_NAMF_COMM, NfServiceStatus: models.Nrf_NFMgmt_NFServiceStatus_REGISTERED, Scheme: models.UriScheme("ftp"), IpEndPoints: []models.Nrf_NFMgmt_IpEndPoint{{Ipv4Address: "10.1.2.3", Port: 8000}}}
		_, err := resolveNFServiceURI(models.Nrf_NFDisc_NFProfile{NfServices: []models.Nrf_NFDisc_NFService{svc}}, models.Nrf_NFMgmt_ServiceName_NAMF_COMM, models.Nrf_NFMgmt_NFServiceStatus_REGISTERED)
		if err == nil {
			t.Fatal("expected bad scheme error")
		}
	})
}
