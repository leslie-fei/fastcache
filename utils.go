package fastcache

import (
	"math/big"
	"unsafe"
)

// NodeTo node data convert to *T
func nodeTo[T any](node *dataNode) *T {
	dataPtr := uintptr(unsafe.Pointer(node)) + sizeOfDataNode
	return (*T)(unsafe.Pointer(dataPtr))
}

func toDataNode(all *allocator, offset uint64) *dataNode {
	if offset == 0 {
		return nil
	}
	return (*dataNode)(unsafe.Pointer(all.base() + uintptr(offset)))
}

func nextPrime(n int) int {
	for {
		if big.NewInt(int64(n)).ProbablyPrime(0) {
			return n
		}
		n++
	}
}

func b2s(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

func s2b(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
