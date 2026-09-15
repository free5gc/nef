package context

import (
	"sync"
	"testing"
)

func TestGetOrCreateAfConcurrent(t *testing.T) {
	nefCtx := &NefContext{afs: make(map[string]*AfData)}

	const goroutineCount = 32
	start := make(chan struct{})
	results := make(chan *AfData, goroutineCount)
	var wg sync.WaitGroup
	for range goroutineCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results <- nefCtx.GetOrCreateAf("af1")
		}()
	}

	close(start)
	wg.Wait()
	close(results)

	var first *AfData
	for af := range results {
		if first == nil {
			first = af
			continue
		}
		if af != first {
			t.Fatal("GetOrCreateAf returned different AF instances for the same ID")
		}
	}

	if first == nil {
		t.Fatal("GetOrCreateAf returned no AF instance")
	}
	if got := nefCtx.GetAf("af1"); got != first {
		t.Fatal("GetOrCreateAf did not register the returned AF instance")
	}
}
