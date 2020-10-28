package nef

import (
)

type NefApp struct {
}

func NewNEF(nefcfgPath string) *NefApp {
	nef := &NefApp{}
	return nef
}

func (n *NefApp) Run() error {
	return nil
}
