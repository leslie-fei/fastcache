package fastcache

import (
	"runtime"
	"sync"
	"sync/atomic"
)

type Locker interface {
	sync.Locker
	RLock()
	RUnlock()
	Reset()
}

type threadLocker struct {
	sync.RWMutex
}

//func (l *threadLocker) Lock() {
//
//}
//
//func (l *threadLocker) Unlock() {
//}

func (l *threadLocker) Reset() {
	// ignore
}

type processLocker struct {
	write    *int32
	read     *int32
	filepath string // file lock
}

func (l *processLocker) Lock() {
	// TODO lock timeout
	for !atomic.CompareAndSwapInt32(l.write, 0, 1) {
		runtime.Gosched()
	}
}

func (l *processLocker) Unlock() {
	if !atomic.CompareAndSwapInt32(l.write, 1, 0) {
		panic("unlock an unlocked-lock")
	}
}

func (l *processLocker) RLock() {
	// TODO read lock
	l.Lock()
}

func (l *processLocker) RUnlock() {
	l.Unlock()
}

func (l *processLocker) Reset() {
	*l.write = 0
}

var nopLocker = &nonLocker{}

type nonLocker struct {
}

func (n *nonLocker) Lock() {
}

func (n *nonLocker) Unlock() {
}

func (n *nonLocker) RLock() {
}

func (n *nonLocker) RUnlock() {
}

func (n *nonLocker) Reset() {
}
