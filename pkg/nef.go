package nef

import (
	"bitbucket.org/free5gc-team/nef/internal/sbi"
)

type NefApp struct {
	sbiServer *sbi.SBIServer
}

func NewNEF(nefcfgPath string) *NefApp {
	nef := &NefApp{}
	nef.sbiServer = sbi.NewSBIServer()
	if nef.sbiServer == nil {
		return nil
	}
	return nef
}

func (n *NefApp) Run() error {
	return n.sbiServer.ListenAndServe("http")
}
