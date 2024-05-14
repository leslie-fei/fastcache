package memlru

import (
	"reflect"
	"unsafe"
)

// DataNode 用来存储对应数据的数据链表节点
type DataNode struct {
	Next           uint64
	Len            uint32
	DataOffset     uint64
	FreeBlockIndex uint8
}

func (ln *DataNode) Reset() {
	ln.Next = 0
	ln.Len = 0
}

func (ln *DataNode) Data(base uintptr) []byte {
	var ss []byte
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&ss))
	sh.Data = base + uintptr(ln.DataOffset)
	sh.Len = int(ln.Len)
	sh.Cap = sh.Len
	return ss
}

func (ln *DataNode) DataPtr(base uintptr) uintptr {
	return uintptr(ln.DataOffset) + base
}

func (ln *DataNode) Offset(base uintptr) uint64 {
	return uint64(uintptr(unsafe.Pointer(ln)) - base)
}

func (ln *DataNode) UpdateData(base uintptr, value []byte) {
	ptr := ln.DataPtr(base)
	// move value to DataNode data ptr
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&value))
	memmove(unsafe.Pointer(ptr), unsafe.Pointer(bh.Data), uintptr(len(value)))
	ln.Len = uint32(len(value))
}

func (ln *DataNode) UpdateString(base uintptr, value string) {
	ptr := ln.DataPtr(base)
	ss := (*reflect.StringHeader)(unsafe.Pointer(&value))
	memmove(unsafe.Pointer(ptr), unsafe.Pointer(ss.Data), uintptr(ss.Len))
	ln.Len = uint32(ss.Len)
}
