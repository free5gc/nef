package nef

import (
	"bitbucket.org/free5gc-team/nef/internal/context"
	"bitbucket.org/free5gc-team/nef/internal/processor"
	"bitbucket.org/free5gc-team/nef/internal/sbi"
)

type NefApp struct {
	nefCtx    *context.NefContext
	processor *processor.Processor
	sbiServer *sbi.SBIServer
}

func NewNEF(nefcfgPath string) *NefApp {
	nef := &NefApp{}
	if nef.nefCtx = context.NewNefContext(); nef.nefCtx == nil {
		return nil
	}
	if nef.processor = processor.NewProcessor(nef.nefCtx); nef.processor == nil {
		return nil
	}
	if nef.sbiServer = sbi.NewSBIServer(nef.processor); nef.sbiServer == nil {
		return nil
	}
	return nef
}

func (n *NefApp) Run() error {
	return n.sbiServer.ListenAndServe("http")
}
