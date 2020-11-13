package context

import (
	"sync"
)

type AfPfdTransaction struct {
	transID string
	mtx     sync.RWMutex
}
