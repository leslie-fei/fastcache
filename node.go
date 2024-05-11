package memlru

import (
	"reflect"
	"unsafe"
)

// LinkedNode 用来存储对应数据的数据链表节点
type LinkedNode struct {
	Next       uint64
	Len        uint32
	DataOffset uint64
	FreeType   int8
}

func (ln *LinkedNode) Reset() {
	ln.Next = 0
	ln.Len = 0
	ln.DataOffset = 0
}

func (ln *LinkedNode) Data(base uintptr) []byte {
	var ss []byte
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&ss))
	sh.Data = base + uintptr(ln.DataOffset)
	sh.Len = int(ln.Len)
	sh.Cap = sh.Len
	return ss
}

func (ln *LinkedNode) DataPtr(base uintptr) uintptr {
	return uintptr(ln.DataOffset) + base
}

func (ln *LinkedNode) Offset(base uintptr) uint64 {
	return uint64(uintptr(unsafe.Pointer(ln)) - base)
}

func (ln *LinkedNode) UpdateData(base uintptr, value []byte) {
	ptr := ln.DataPtr(base)
	// move value to LinkedNode data ptr
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&value))
	memmove(unsafe.Pointer(ptr), unsafe.Pointer(bh.Data), uintptr(len(value)))
	ln.Len = uint32(len(value))
}
