package fastcache

import (
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

var sizeOfProcessLocker = unsafe.Sizeof(processLocker{})

type Locker interface {
	sync.Locker
}

type processLocker struct {
	write int32
	read  int32
}

func (l *processLocker) Lock() {
	// TODO lock timeout
	for !atomic.CompareAndSwapInt32(&l.write, 0, 1) {
		runtime.Gosched()
	}
}

func (l *processLocker) Unlock() {
	if !atomic.CompareAndSwapInt32(&l.write, 1, 0) {
		panic("unlock an unlocked-lock")
	}
}

func (l *processLocker) Reset() {
	l.write = 0
	l.read = 0
}

type nopLocker struct{}

func (n *nopLocker) Lock() {
}

func (n *nopLocker) Unlock() {
}
