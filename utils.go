package memlru

import "unsafe"

// NodeConvertTo node data convert to *T
func NodeConvertTo[T any](base uintptr, node *LinkedNode) *T {
	return (*T)(unsafe.Pointer(node.DataPtr(base)))
}

func RangeNode(base uintptr, offset uint64, f func(node *LinkedNode) bool) {
	if offset == 0 {
		return
	}
	for node := ToLinkedNode(base, offset); node != nil; node = ToLinkedNode(base, offset) {
		if !f(node) {
			return
		}
	}
}

func ToLinkedNode(base uintptr, offset uint64) *LinkedNode {
	if offset == 0 {
		return nil
	}
	return (*LinkedNode)(unsafe.Pointer(base + uintptr(offset)))
}
