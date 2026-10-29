package consumer

import (
	"errors"
	"fmt"
	"log"
	"sort"

	"github.com/free5gc/openapi/models"
	"github.com/free5gc/util/serviceuri"
)

var errNFServiceAddressUnavailable = serviceuri.ErrAddressUnavailable

type nfServiceURIOptions = serviceuri.Options

func convertNFServiceEndpoints(points []models.Nrf_NFMgmt_IpEndPoint) []serviceuri.Endpoint {
	out := make([]serviceuri.Endpoint, 0, len(points))
	for _, p := range points {
		out = append(out, serviceuri.Endpoint{IPv4Address: p.Ipv4Address, IPv6Address: p.Ipv6Address, Port: p.Port})
	}
	return out
}

func buildNFServiceURI(profile models.Nrf_NFDisc_NFProfile,
	service models.Nrf_NFDisc_NFService, opts nfServiceURIOptions,
) (string, error) {
	result, err := serviceuri.Build(
		serviceuri.Profile{
			FQDN:          profile.Fqdn,
			IPv4Addresses: profile.Ipv4Addresses,
			IPv6Addresses: profile.Ipv6Addresses,
		},
		serviceuri.Service{
			Scheme:    string(service.Scheme),
			FQDN:      service.Fqdn,
			APIPrefix: service.ApiPrefix,
			Endpoints: convertNFServiceEndpoints(service.IpEndPoints),
		}, opts,
	)
	if err == nil && result.UsedLegacyAbsoluteAPIPrefix {
		log.Printf("Deprecated absolute NFService.apiPrefix accepted for compatibility: %q", service.ApiPrefix)
	}
	return result.APIRoot, err
}

func validateNFServiceIPPoint(p models.Nrf_NFMgmt_IpEndPoint) error {
	return serviceuri.ValidateIPPoint(serviceuri.Endpoint{
		IPv4Address: p.Ipv4Address, IPv6Address: p.Ipv6Address, Port: p.Port,
	})
}

func portForNFServiceFQDN(points []models.Nrf_NFMgmt_IpEndPoint) (int32, error) {
	return serviceuri.PortForFQDN(convertNFServiceEndpoints(points))
}

func validateNFServiceAPIPrefix(prefix string) error {
	return serviceuri.ValidateAPIPrefix(prefix)
}

func buildNFServiceAPIRoot(scheme, host string, port int32, prefix string) (string, error) {
	return serviceuri.BuildAPIRoot(scheme, host, port, prefix)
}

// resolveNFServiceURI selects a registered NFService and returns
// its complete API root; legacy absolute apiPrefix is transitional only.
func resolveNFServiceURI(profile models.Nrf_NFDisc_NFProfile,
	name models.Nrf_NFMgmt_ServiceName, status models.Nrf_NFMgmt_NFServiceStatus) (string, error) {
	var last error
	for _, service := range profile.NfServices {
		if service.ServiceName != name || service.NfServiceStatus != status {
			continue
		}
		uri, err := buildNFServiceURI(profile, service, nfServiceURIOptions{AllowLegacyAbsoluteAPIPrefix: true})
		if err == nil {
			return uri, nil
		}
		if !errors.Is(err, errNFServiceAddressUnavailable) {
			return "", err
		}
		last = err
	}
	keys := make([]string, 0, len(profile.NfServiceList))
	for key := range profile.NfServiceList {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		service := profile.NfServiceList[key]
		if service.ServiceName != name || service.NfServiceStatus != status {
			continue
		}
		uri, err := buildNFServiceURI(profile, service, nfServiceURIOptions{AllowLegacyAbsoluteAPIPrefix: true})
		if err == nil {
			return uri, nil
		}
		if !errors.Is(err, errNFServiceAddressUnavailable) {
			return "", err
		}
		last = err
	}
	if last != nil {
		return "", last
	}
	return "", fmt.Errorf("NF service %q with status %q not found", name, status)
}
