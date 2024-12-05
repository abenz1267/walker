package util

import (
	"sync"
	"time"
)

func NewDebounce(after time.Duration) func(f func()) {
	d := &Debouncer{after: after}

	return func(f func()) {
		d.add(f)
	}
}

type Debouncer struct {
	mu    sync.Mutex
	after time.Duration
	timer *time.Timer
}

func (d *Debouncer) add(f func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
	}

	d.timer = time.AfterFunc(d.after, f)
}
