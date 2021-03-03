package number_pool

import (
	"sync/atomic"
	"time"
)

type NumberPool struct {
	numberArr []uint64
	number uint64
	currentNum int64
	maxVal uint64
	add uint64
}

/**
 * 创建一个编号池
 * @param		maxVal, add uint64		最大编号, 每次增加值
 * @return		*NumberPool				编号池对象的指针
 * func NewNumberPool(maxVal, add uint64) *NumberPool;
 */
func NewNumberPool(maxVal, add uint64) *NumberPool {
	p := &NumberPool{
		numberArr:make([]uint64, maxVal+1),
		number: 1,
		maxVal: maxVal,
		add:    add,
	}
	go func() {
		for {
			time.Sleep(5 * time.Second)
			num := 0
			for i := 0;i < int(maxVal);i++ {
				if p.numberArr[i] == 0 {
					num++
				}
			}
		}
	}()
	return p
}

/**
 * 从编号池中取出一个未使用的编号
 * @param		nil
 * @return		uint64, bool	编号, 是否可取
 * func (n *NumberPool)Get() (uint64, bool);
 */
func (n *NumberPool)Get() (uint64, bool) {
	if uint64(atomic.LoadInt64(&n.currentNum)) > n.maxVal {
		return 0, false
	}
	if uint64(atomic.AddInt64(&n.currentNum, 1)) > n.maxVal {
		atomic.AddInt64(&n.currentNum, -1)
		return 0, false
	}
	num := 0
	for i := atomic.LoadUint64(&n.number);;i = atomic.AddUint64(&n.number, n.add) {
		atomic.CompareAndSwapUint64(&n.number, n.maxVal, 1)
		num++
		if num / int(n.maxVal) >= 3 {
			atomic.AddInt64(&n.currentNum, -1)
			return 0, false
		}
		if i > n.maxVal {
			i = 1
		}
		if atomic.CompareAndSwapUint64(&n.numberArr[i], 0, 1) {
			return i, true
		}
	}

	atomic.AddInt64(&n.currentNum, -1)
	return 0, false
}

/**
 * 将编号放入编号池中
 * @param		number int		编号
 * @return		nil
 * func (n *NumberPool)Put(number int);
 */
func (n *NumberPool)Put(number int) {
	atomic.AddInt64(&n.currentNum, -1)
	atomic.CompareAndSwapUint64(&n.numberArr[number], 1, 0)
}
