package ui

import (
	"sync"
	"time"

	"github.com/abenz1267/walker/internal/util"
)

type PopulateListParams struct {
	Text                     string
	KeepSort                 bool
	ProcessedModulesKeepSort []bool
	Entries                  []*util.Entry
}

type LatestOnlyThrottler struct {
	ticker     *time.Ticker
	latestCall *PopulateListParams
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
			if t.hasCall && t.latestCall != nil {
				params := *t.latestCall
				t.hasCall = false
				t.mu.Unlock()

				populateList(params.Text, params.KeepSort, params.ProcessedModulesKeepSort, params.Entries)
			} else {
				t.mu.Unlock()
			}
		}
	}
}

func (t *LatestOnlyThrottler) Execute(text string, keepSort bool, processedModulesKeepSort []bool, entries []*util.Entry) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Always replace with the latest call
	t.latestCall = &PopulateListParams{
		Text:                     text,
		KeepSort:                 keepSort,
		ProcessedModulesKeepSort: processedModulesKeepSort,
		Entries:                  entries,
	}
	t.hasCall = true
}

func (t *LatestOnlyThrottler) Stop() {
	t.once.Do(func() {
		close(t.stop)
		t.ticker.Stop()
	})
}
