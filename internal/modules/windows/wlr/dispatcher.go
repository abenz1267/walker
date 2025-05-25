package wlr

var (
	addChan       chan string
	deleteChan    chan string
	addChanSub    []chan string
	deleteChanSub []chan string
)

func init() {
	addChan = make(chan string)
	deleteChan = make(chan string)

	go func() {
		for {
			select {
			case appId := <-addChan:
				for _, v := range addChanSub {
					v <- appId
				}
			case appId := <-deleteChan:
				for _, v := range deleteChanSub {
					v <- appId
				}
			}
		}
	}()
}

func Subscribe(add chan string, delete chan string) {
	addChanSub = append(addChanSub, add)
	deleteChanSub = append(deleteChanSub, delete)
}
