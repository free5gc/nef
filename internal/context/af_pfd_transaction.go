package context

import (
	"sync"
)

type AfPfdTransaction struct {
	transID        string
	externalAppIDs map[string]bool
	mtx            sync.RWMutex
}

func (a *AfPfdTransaction) GetTransID() string {
	a.mtx.RLock()
	defer a.mtx.RUnlock()
	return a.transID
}

func (a *AfPfdTransaction) GetExtAppIDs() []string {
	a.mtx.RLock()
	defer a.mtx.RUnlock()
	ids := make([]string, 0, len(a.externalAppIDs))
	for id := range a.externalAppIDs {
		ids = append(ids, id)
	}
	return ids
}

func (a *AfPfdTransaction) IsAppIDExisted(appID string) bool {
	a.mtx.RLock()
	defer a.mtx.RUnlock()
	return a.externalAppIDs[appID]
}

func (a *AfPfdTransaction) AddExtAppID(appID string) {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	a.externalAppIDs[appID] = true
}

func (a *AfPfdTransaction) DeleteExtAppID(appID string) {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	delete(a.externalAppIDs, appID)
}

func (a *AfPfdTransaction) DeleteAllExtAppIDs() {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	a.externalAppIDs = make(map[string]bool)
}
