package fastcache

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

func TestUtils(t *testing.T) {
	allocator := mockAllocator
	container := &lruAndFreeContainer{}
	container.Init(allocator.Base())

	type testStruct struct {
		v uint8
	}

	node, err := container.Alloc(allocator, uint64(unsafe.Sizeof(testStruct{})))
	if err != nil {
		panic(err)
	}

	node.Len = uint32(unsafe.Sizeof(testStruct{}))
	ts := NodeTo[testStruct](node)
	ts.v = 1

	offset := uintptr(unsafe.Pointer(node)) - allocator.Base()
	node = ToDataNode(allocator.Base(), uint64(offset))

	nts := NodeTo[testStruct](node)
	assert.Equal(t, ts.v, nts.v)

	node = ToDataNode(allocator.Base(), 0)
	assert.Nil(t, node)
}
