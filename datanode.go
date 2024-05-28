package fastcache

import "unsafe"

var sizeOfDataNode = unsafe.Sizeof(dataNode{})

type dataNode struct {
	next      uint64
	freeIndex uint8
}

func (d *dataNode) reset() {
	*d = dataNode{}
}

func (d *dataNode) offset(all *allocator) uint64 {
	ptr := uintptr(unsafe.Pointer(d))
	return uint64(ptr - all.base())
}
