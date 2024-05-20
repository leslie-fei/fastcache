package fastcache

import (
	"unsafe"
)

// DataNode 用来存储对应数据的数据链表节点
type DataNode struct {
	Next           uint64
	Len            uint32
	FreeBlockIndex uint8
}

func (ln *DataNode) Reset() {
	ln.Next = 0
	ln.Len = 0
}

func (ln *DataNode) Offset(base uintptr) uint64 {
	return uint64(uintptr(unsafe.Pointer(ln)) - base)
}
