package consumer

import (
	"bitbucket.org/free5gc-team/nef/internal/context"
	"bitbucket.org/free5gc-team/nef/internal/factory"
	"bitbucket.org/free5gc-team/openapi/Nudr_DataRepository"
)

type ConsumerUDRService struct {
	cfg            *factory.Config
	nefCtx         *context.NefContext
	clientDataRepo *Nudr_DataRepository.APIClient
}

func NewConsumerUDRService(nefCfg *factory.Config, nefCtx *context.NefContext) *ConsumerUDRService {
	c := &ConsumerUDRService{cfg: nefCfg, nefCtx: nefCtx}

	drConfig := Nudr_DataRepository.NewConfiguration()
	drConfig.SetBasePath(c.nefCtx.GetUdrURI())
	c.clientDataRepo = Nudr_DataRepository.NewAPIClient(drConfig)
	return c
}
