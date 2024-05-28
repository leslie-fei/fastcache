package fastcache

import (
	"unsafe"
)

// allocator 全局的内存分配, 所有的内存分配最终都是通过他分配出去
type allocator struct {
	mem      Memory
	metadata *metadata
	locker   Locker
}

func (g *allocator) alloc(size uint64) (ptr unsafe.Pointer, offset uint64, err error) {
	g.locker.Lock()
	defer g.locker.Unlock()
	if size > g.freeMemory() {
		err = ErrNoSpace
		return
	}
	offset = g.metadata.Used
	ptr = g.mem.PtrOffset(offset)
	g.metadata.Used += size
	return
}

func (g *allocator) freeMemory() uint64 {
	return g.metadata.TotalSize - g.metadata.Used
}

func (g *allocator) base() uintptr {
	return uintptr(g.mem.Ptr())
}

func (g *allocator) offset() uint64 {
	return g.metadata.Used
}

func (g *allocator) setLocker(locker Locker) {
	g.locker = locker
}
