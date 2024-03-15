package weak

import "sync/atomic"

type refState uint32

const (
	refDEAD  refState = 0
	refALIVE refState = 1
	refINUSE refState = 2
)

func (wrS *refState) CaS(old, new refState) bool {
	return atomic.CompareAndSwapUint32((*uint32)(wrS), uint32(old), uint32(new))
}

func (wrS *refState) Set(v refState) {
	atomic.StoreUint32((*uint32)(wrS), uint32(v))
}

func (wrS *refState) Get() refState {
	return refState(atomic.LoadUint32((*uint32)(wrS)))
}
