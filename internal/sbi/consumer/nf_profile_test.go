package consumer

import (
	"testing"

	"github.com/free5gc/nef/pkg/factory"
	"github.com/stretchr/testify/require"
)

// TestBuildNfProfileFromSampleConfig guards against adding a service to the
// sample config without an AllowedNfTypes policy, which would make NRF
// registration (and therefore NEF startup) fail.
func TestBuildNfProfileFromSampleConfig(t *testing.T) {
	cfg, err := factory.ReadConfig("../../../config/nefcfg.yaml")
	require.NoError(t, err)

	profile, err := newTestConsumer(t, cfg).buildNfProfile()
	require.NoError(t, err)
	require.Len(t, profile.NfServices, len(cfg.Configuration.ServiceList))
}
