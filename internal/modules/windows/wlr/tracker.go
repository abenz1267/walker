package wlr

var (
	addChan     chan string
	deleteChan  chan string
	OpenWindows = make(map[string]uint)
)

func init() {
	addChan = make(chan string)
	deleteChan = make(chan string)

	go func() {
		for {
			select {
			case appId := <-addChan:
				if _, ok := OpenWindows[appId]; ok {
					OpenWindows[appId] = OpenWindows[appId] + 1
				} else {
					OpenWindows[appId] = 1
				}
			case appId := <-deleteChan:
				if val, ok := OpenWindows[appId]; ok {
					if val == 1 {
						delete(OpenWindows, appId)
					} else {
						OpenWindows[appId] = val - 1
					}
				}
			}
		}
	}()
}
