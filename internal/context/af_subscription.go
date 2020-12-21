package context

import (
	"sync"
)

type AfSubscription struct {
	subscID            string
	appSessID          string //use in single UE case
	influID            string //use in multiple UE case
	notifCorreID       string
	notificationURI    string
	isIndividualUEAddr bool // false in UDR, true in PCF
	mtx                sync.RWMutex
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

func (s *AfSubscription) SetAppSessID(id string) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.appSessID = id
}

func (s *AfSubscription) SetInfluenceID(id string) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.influID = id
}

func (s *AfSubscription) SetNotificationURI(uri string) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.notificationURI = uri
}

func (s *AfSubscription) GetAppSessID() string {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	return s.appSessID
}

func (s *AfSubscription) GetInfluenceID() string {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	return s.influID
}

func (s *AfSubscription) GetIsIndividualUEAddr() bool {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s.isIndividualUEAddr
}
