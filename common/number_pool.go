package common

import (
	"sync/atomic"
)

type NumberPool struct {
	numberArr []uint64
	number uint64
	maxVal uint64
	add uint64
}

func NewNumberPool(val, add uint64) *NumberPool {
	return &NumberPool{
		numberArr:make([]uint64, val+1),
		number: 1,
		maxVal: val,
		add:    add,
	}
}

func (n *NumberPool)Get() (uint64, bool) {
	num := 0
	for i := atomic.LoadUint64(&n.number);;i = atomic.AddUint64(&n.number, n.add) {
		atomic.CompareAndSwapUint64(&n.number, n.maxVal, 1)
		num++
		if num / int(n.maxVal) >= 3 {
			return 0, false
		}
		if i > n.maxVal {
			i = 1
		}
		if atomic.CompareAndSwapUint64(&n.numberArr[i], 0, 1) {
			return i, true
		}
	}
	return 0, false
}

func (n *NumberPool)Put(v int) {
	atomic.CompareAndSwapUint64(&n.numberArr[v], 1, 0)
}