package context

import (
	"sync"
)

type AfSubscription struct {
	subscID         string
	appSessID       string //use in single UE case
	influID         string //use in multiple UE case
	notifCorreID    string
	notificationURI string
	mtx             sync.RWMutex
}

func (s *AfSubscription) GetSubscID() string {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s.subscID
}

func (s *AfSubscription) GetNotifCorreID() string {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s.notifCorreID
}

func (s *AfSubscription) AppSessID(id string) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.appSessID = id
}

func (s *AfSubscription) InfluenceID(id string) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.influID = id
}

func (s *AfSubscription) NotificationURI(uri string) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.notificationURI = uri
}
