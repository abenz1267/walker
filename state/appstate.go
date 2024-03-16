package state

import "time"

type AppState struct {
	Started    time.Time
	IsMeasured bool
	IsService  bool
	IsRunning  bool
}

func Get() *AppState {
	return &AppState{
		Started:    time.Now(),
		IsService:  false,
		IsRunning:  false,
		IsMeasured: false,
	}
}
