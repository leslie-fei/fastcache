package memlru

import "unsafe"

// NodeConvertTo node data convert to *T
func NodeConvertTo[T any](base uintptr, node *DataNode) *T {
	return (*T)(unsafe.Pointer(node.DataPtr(base)))
}

func RangeNode(base uintptr, offset uint64, f func(node *DataNode) bool) {
	if offset == 0 {
		return
	}
	for node := ToDataNode(base, offset); node != nil; node = ToDataNode(base, offset) {
		if !f(node) {
			return
		}
	}
}

func ToDataNode(base uintptr, offset uint64) *DataNode {
	if offset == 0 {
		return nil
	}
	return (*DataNode)(unsafe.Pointer(base + uintptr(offset)))
}
