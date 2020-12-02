package context

import (
	"sync"
)

type AfPfdTransaction struct {
	transID string
	mtx     sync.RWMutex
}

func (a *AfPfdTransaction) GetTransID() string {
	a.mtx.RLock()
	defer a.mtx.RUnlock()
	return a.transID
}
