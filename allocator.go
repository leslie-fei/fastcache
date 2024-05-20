package fastcache

import (
	"errors"
	"unsafe"
)

var sizeOfShardMemory = unsafe.Sizeof(shardMemory{})
var ErrAllocSizeTooLarge = errors.New("alloc size too large")

type Allocator interface {
	Alloc(size uint64) (ptr unsafe.Pointer, offset uint64, err error)
	Base() uintptr
	Locker() Locker
}

// globalAllocator 全局的内存分配, 所有的内存分配最终都是通过他分配出去
type globalAllocator struct {
	mem      Memory
	metadata *Metadata
}

func (g *globalAllocator) Alloc(size uint64) (ptr unsafe.Pointer, offset uint64, err error) {
	if size > g.FreeMemory() {
		err = ErrNoSpace
		return
	}
	ptr = g.mem.PtrOffset(g.metadata.Used)
	offset = g.metadata.Used
	g.metadata.Used += size
	return
}

func (g *globalAllocator) FreeMemory() uint64 {
	return g.metadata.TotalSize - g.metadata.Used
}

func (g *globalAllocator) Base() uintptr {
	return uintptr(g.mem.Ptr())
}

func (g *globalAllocator) Locker() Locker {
	return g.metadata.GlobalLocker
}

type shardMemory struct {
	offset uint64
	size   uint64
	used   uint64
	next   uint64
}

func (s *shardMemory) Reset() {
	s.size = 0
	s.offset = 0
	s.used = 0
	s.next = 0
}

func (s *shardMemory) FreeMemory() uint64 {
	return s.size - s.used
}

// shardAllocator 归属于shard的内存分配, 他们都先从 globalAllocator 中分配出内存, 给shard独享
type shardAllocator struct {
	global         *globalAllocator
	first          uint64
	shardMemoryLen uint32
	growSize       uint64
}

func (s *shardAllocator) Alloc(size uint64) (ptr unsafe.Pointer, offset uint64, err error) {
	if size > s.growSize-uint64(sizeOfShardMemory) {
		err = ErrAllocSizeTooLarge
		return
	}
	mem := s.findShardMemory(size)
	if mem == nil {
		// 当shardMemory没有可以分配出size大小的块, 就需要去globalAllocator中申请
		globalLocker := s.global.Locker()
		globalLocker.Lock()
		ptr, offset, err = s.global.Alloc(s.growSize)
		globalLocker.Unlock()
		if err != nil {
			return
		}
		newMem := (*shardMemory)(ptr)
		newMem.Reset()
		newMem.offset = offset
		newMem.size = s.growSize - uint64(sizeOfShardMemory)
		if s.first == 0 {
			s.first = newMem.offset
		} else {
			next := s.first
			s.first = newMem.offset
			newMem.next = next
		}
		mem = newMem
	}

	offset = mem.offset + mem.used
	ptr = unsafe.Pointer(s.Base() + uintptr(offset))
	mem.used += size
	return
}

func (s *shardAllocator) Base() uintptr {
	return s.global.Base()
}

func (s *shardAllocator) Locker() Locker {
	return nopLocker
}

// findShardMemory 找到一个可用的shardMemory
func (s *shardAllocator) findShardMemory(size uint64) *shardMemory {
	offset := s.first
	for i := 0; i < int(s.shardMemoryLen); i++ {
		mem := (*shardMemory)(unsafe.Pointer(s.Base() + uintptr(offset)))
		if mem.FreeMemory() >= size {
			return mem
		}
		offset = mem.next
	}
	return nil
}
