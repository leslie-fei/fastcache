package memlru

import (
	"runtime"
	"sync/atomic"
)

type Locker struct {
	lock int32
}

func (l *Locker) Lock() {
	for !atomic.CompareAndSwapInt32(&l.lock, 0, 1) {
		runtime.Gosched()
	}
}

func (l *Locker) Unlock() {
	if !atomic.CompareAndSwapInt32(&l.lock, 1, 0) {
		panic("unlock an unlocked-lock")
	}
}

func (l *Locker) RLock() {
	// TODO read lock
	l.Lock()
}

func (l *Locker) RUnlock() {
	l.Unlock()
}

func (l *Locker) Reset() {
	l.lock = 0
}
