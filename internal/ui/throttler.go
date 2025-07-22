package ui

import (
	"sync"
	"time"
)

type LatestOnlyThrottler struct {
	ticker     *time.Ticker
	latestCall int
	hasCall    bool
	mu         sync.Mutex
	stop       chan struct{}
	once       sync.Once
}

func NewLatestOnlyThrottler(interval time.Duration) *LatestOnlyThrottler {
	t := &LatestOnlyThrottler{
		ticker: time.NewTicker(interval),
		stop:   make(chan struct{}),
	}

	go t.run()
	return t
}

func (t *LatestOnlyThrottler) run() {
	for {
		select {
		case <-t.stop:
			return
		case <-t.ticker.C:
			t.mu.Lock()
			if t.hasCall && t.latestCall != 0 {
				params := t.latestCall
				t.hasCall = false
				t.mu.Unlock()

				appstate.DmenuStreamAdded <- params
			} else {
				t.mu.Unlock()
			}
		}
	}
}

func (t *LatestOnlyThrottler) Execute(id int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.latestCall = id
	t.hasCall = true
}

func (t *LatestOnlyThrottler) Stop() {
	t.once.Do(func() {
		close(t.stop)
		t.ticker.Stop()
	})
}
