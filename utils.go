package fastcache

import (
	"math/big"
	"runtime"
	"strconv"
	"strings"
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

func getGoroutineID() uint64 {
	buf := make([]byte, 64)
	runtime.Stack(buf, false)
	for _, f := range strings.Split(string(buf), "\n") {
		if strings.Contains(f, "goroutine ") {
			ids := strings.Split(f, " ")
			if len(ids) > 1 {
				id, _ := strconv.ParseUint(ids[1], 10, 64)
				return id
			}
		}
	}
	return 0
}
