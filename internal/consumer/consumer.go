package consumer

import (
	"bitbucket.org/free5gc-team/nef/internal/context"
	"bitbucket.org/free5gc-team/nef/internal/factory"
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
	if c.PcfSrv = NewConsumerPCFService(nefCfg, nefCtx, c.NrfSrv); c.PcfSrv == nil {
		return nil
	}
	if c.UdrSrv = NewConsumerUDRService(nefCfg, nefCtx, c.NrfSrv); c.UdrSrv == nil {
		return nil
	}
	c.NrfSrv.RegisterNFInstance()

	return c
}
