package consumer

import (
	"bitbucket.org/free5gc-team/nef/internal/context"
	"bitbucket.org/free5gc-team/nef/internal/factory"
	"bitbucket.org/free5gc-team/nef/internal/logger"
)

type Consumer struct {
	NrfSrv *ConsumerNRFService
	PcfSrv *ConsumerPCFService
	UdrSrv *ConsumerUDRService
}

func NewConsumer(nefCfg *factory.Config, nefCtx *context.NefContext) *Consumer {
	c := &Consumer{}
	if c.NrfSrv = NewConsumerNRFService(nefCfg, nefCtx); c.NrfSrv == nil {
		return nil
	}
	c.NrfSrv.RegisterNFInstance()
	if uri, err := c.NrfSrv.SearchNFServiceUri("PCF", "npcf-policyauthorization"); err != nil {
		logger.ConsumerLog.Errorf("%+v", err)
		return nil
	} else {
		nefCtx.PcfURI(uri)
		if c.PcfSrv = NewConsumerPCFService(nefCfg, nefCtx); c.PcfSrv == nil {
			return nil
		}
	}
	if uri, err := c.NrfSrv.SearchNFServiceUri("UDR", "nudr-dr"); err != nil {
		logger.ConsumerLog.Errorf("%+v", err)
		return nil
	} else {
		nefCtx.UdrURI(uri)
		if c.UdrSrv = NewConsumerUDRService(nefCfg, nefCtx); c.UdrSrv == nil {
			return nil
		}
	}

	return c
}
