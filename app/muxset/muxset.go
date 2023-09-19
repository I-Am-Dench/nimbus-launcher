package muxset

import "sync"

var exists = struct{}{}

type MuxSet[K comparable] struct {
	data map[K]struct{}
	mux  sync.Mutex
}

func New[K comparable]() *MuxSet[K] {
	set := new(MuxSet[K])
	set.data = make(map[K]struct{})
	return set
}

func (set *MuxSet[K]) Add(item K) {
	set.mux.Lock()
	defer set.mux.Unlock()
	set.data[item] = exists
}

func (set *MuxSet[K]) Delete(item K) {
	set.mux.Lock()
	defer set.mux.Unlock()
	delete(set.data, item)
}

func (set *MuxSet[K]) Has(item K) bool {
	set.mux.Lock()
	defer set.mux.Unlock()
	_, ok := set.data[item]
	return ok
}
