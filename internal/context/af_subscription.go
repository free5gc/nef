package context

import (
	"sync"

	"github.com/free5gc/openapi/models"
	"github.com/sirupsen/logrus"
)

type AfSubscription struct {
	SubID        string
	TiSub        *models.Nef_TrafInfl_TrafficInfluSub
	AppSessID    string // use in single UE case
	InfluID      string // use in multiple UE case
	NotifCorreID string
	Log          *logrus.Entry

	// OpMu serializes outbound mutations for this subscription.
	// It must be acquired without holding the parent AfData.Mu.
	OpMu sync.Mutex
}

func (s *AfSubscription) PatchTiSubData(tiSubPatch *models.Nef_TrafInfl_TrafficInfluSubPatch) {
	s.TiSub.AppReloInd = tiSubPatch.AppReloInd
	s.TiSub.TrafficFilters = tiSubPatch.TrafficFilters
	s.TiSub.EthTrafficFilters = tiSubPatch.EthTrafficFilters
	s.TiSub.TrafficRoutes = tiSubPatch.TrafficRoutes
	s.TiSub.TfcCorrInd = tiSubPatch.TfcCorrInd
	s.TiSub.TempValidities = tiSubPatch.TempValidities
	s.TiSub.ValidGeoZoneIds = tiSubPatch.ValidGeoZoneIds // deprecated
	s.TiSub.AfAckInd = tiSubPatch.AfAckInd
	s.TiSub.AddrPreserInd = tiSubPatch.AddrPreserInd
}
