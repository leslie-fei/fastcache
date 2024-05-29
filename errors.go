package fastcache

import "errors"

var (
	ErrNoSpace            = errors.New("memory no space")
	ErrMemorySizeTooSmall = errors.New("memory size too small")
	ErrNotFound           = errors.New("key not found")
	ErrIndexOutOfRange    = errors.New("index out of range")
	ErrFreeListIsEmpty    = errors.New("free list is empty")
	ErrLRUListIsEmpty     = errors.New("lru list is empty")
	ErrCacheClosed        = errors.New("cache closed")
	ErrCloseTimeout       = errors.New("cache close timeout")
)
