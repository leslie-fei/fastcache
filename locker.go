package fastcache

import (
	"sync"
)

var rw sync.RWMutex

type Locker struct {
	lock int32
}

func (l *Locker) Lock() {
	rw.Lock()
	// TODO lock timeout
	/*for !atomic.CompareAndSwapInt32(&l.lock, 0, 1) {
		runtime.Gosched()
	}*/
}

func (l *Locker) Unlock() {
	rw.Unlock()
	//if !atomic.CompareAndSwapInt32(&l.lock, 1, 0) {
	//	panic("unlock an unlocked-lock")
	//}
}

func (l *Locker) RLock() {
	// TODO read lock
	//l.Lock()
	rw.RLock()
}

func (l *Locker) RUnlock() {
	//l.Unlock()
	rw.RUnlock()
}

func (l *Locker) Reset() {
	l.lock = 0
}
