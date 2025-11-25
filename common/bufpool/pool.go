package bufpool

import (
	"sync"
)

// DefaultSize размер буфера по умолчанию (64KB - оптимально для TCP)
const DefaultSize = 64 * 1024

// Pool пул буферов для уменьшения GC pressure
type Pool struct {
	pool sync.Pool
}

// NewPool создает новый пул буферов
func NewPool(size int) *Pool {
	return &Pool{
		pool: sync.Pool{
			New: func() interface{} {
				buf := make([]byte, size)
				return &buf
			},
		},
	}
}

// Get получает буфер из пула
func (p *Pool) Get() []byte {
	bufPtr := p.pool.Get().(*[]byte)
	return *bufPtr
}

// Put возвращает буфер в пул
func (p *Pool) Put(buf []byte) {
	// Очищаем буфер перед возвратом в пул
	for i := range buf {
		buf[i] = 0
	}
	p.pool.Put(&buf)
}

// Global pools
var (
	// SmallPool для маленьких буферов (4KB)
	SmallPool = NewPool(4 * 1024)

	// MediumPool для средних буферов (16KB)
	MediumPool = NewPool(16 * 1024)

	// LargePool для больших буферов (64KB)
	LargePool = NewPool(DefaultSize)

	// HugePool для огромных буферов (128KB)
	HugePool = NewPool(128 * 1024)
)

// Get получает буфер оптимального размера
func Get(size int) []byte {
	switch {
	case size <= 4*1024:
		return SmallPool.Get()[:size]
	case size <= 16*1024:
		return MediumPool.Get()[:size]
	case size <= 64*1024:
		return LargePool.Get()[:size]
	default:
		return HugePool.Get()[:size]
	}
}

// Put возвращает буфер в соответствующий пул
func Put(buf []byte) {
	cap := len(buf)
	switch {
	case cap <= 4*1024:
		SmallPool.Put(buf[:cap])
	case cap <= 16*1024:
		MediumPool.Put(buf[:cap])
	case cap <= 64*1024:
		LargePool.Put(buf[:cap])
	default:
		HugePool.Put(buf[:cap])
	}
}
