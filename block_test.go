package fastcache

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

func TestLruAndFreeContainer_Init(t *testing.T) {
	allocator := mockAllocator
	container := &lruAndFreeContainer{}
	container.Init(allocator.Base())

	assert.Equal(t, FreeListLen, container.Len())
	for i, freeList := range container.freeLists {
		assert.Equal(t, uint64(1<<i), freeList.Size)
		assert.Equal(t, uint8(i), freeList.Index)
	}
}

func TestLruAndFreeContainer_Get(t *testing.T) {
	allocator := mockAllocator
	container := &lruAndFreeContainer{}
	container.Init(allocator.Base())

	_, err := container.Get(0)
	assert.Error(t, err)

	for i := 0; i < FreeListLen; i++ {
		freeList, err := container.Get(uint64(1 << i))
		assert.NoError(t, err)
		assert.Equal(t, uint64(1<<i), freeList.Size)
	}

	_, err = container.Get(uint64(1 << FreeListLen))
	assert.Error(t, err)
}

func TestLruAndFreeContainer_Alloc(t *testing.T) {
	allocator := mockAllocator
	container := &lruAndFreeContainer{}
	container.Init(allocator.Base())

	node, err := container.Alloc(allocator, 1)
	assert.NoError(t, err)
	assert.NotNil(t, node)
	assert.Equal(t, uint8(0), node.FreeBlockIndex)
	assert.Equal(t, uint64(0), node.Next)

	node, err = container.Alloc(allocator, 1<<FreeListLen)
	assert.Error(t, err)
	assert.Nil(t, node)

	node, err = container.Alloc(allocator, 1<<(FreeListLen-1))
	assert.NoError(t, err)
	assert.NotNil(t, node)
	assert.Equal(t, uint8(24), node.FreeBlockIndex)

	allocator = newMockAllocator(16 * KB)
	node, err = container.Alloc(allocator, MB)
	assert.Error(t, err)
}

func TestLruAndFreeContainer_Free(t *testing.T) {
	allocator := mockAllocator
	container := &lruAndFreeContainer{}
	container.Init(allocator.Base())

	node, err := container.Alloc(allocator, 1)
	assert.NoError(t, err)

	freeList, err := container.Get(1)
	assert.NoError(t, err)
	freeLen := freeList.Len

	lruNode := &listNode{}
	container.Free(allocator.Base(), node, lruNode)
	assert.Equal(t, uint64(freeLen+1), uint64(freeList.Len))
}

func TestLruAndFreeContainer_Evict(t *testing.T) {
	allocator := mockAllocator
	container := &lruAndFreeContainer{}
	container.Init(allocator.Base())

	node, err := container.Alloc(allocator, 1)
	assert.NoError(t, err)

	lruNode := &listNode{}
	container.PushFront(allocator.Base(), node, lruNode)

	err = container.Evict(allocator, 1, func(node *listNode) {})
	assert.NoError(t, err)

	err = container.Evict(allocator, 1, func(node *listNode) {})
	assert.Error(t, err)
}

func TestLruAndFreeContainer_MoveToFront(t *testing.T) {
	allocator := mockAllocator
	container := &lruAndFreeContainer{}
	container.Init(allocator.Base())

	node := &DataNode{FreeBlockIndex: 0}
	lruNode := &listNode{}
	lruList := container.lruLists[0]
	if lruList.Len() == 0 {
		container.PushFront(allocator.Base(), node, lruNode)
	}
	container.MoveToFront(allocator.Base(), node, lruNode)
}

func TestLruAndFreeContainer_PushFront(t *testing.T) {
	allocator := mockAllocator
	container := &lruAndFreeContainer{}
	container.Init(allocator.Base())

	node := &DataNode{FreeBlockIndex: 0}
	lruNode := &listNode{}
	container.PushFront(allocator.Base(), node, lruNode)
}

func TestBlockFreeList_First(t *testing.T) {
	freeList := &blockFreeList{Head: 0, Len: 0}
	base := uintptr(unsafe.Pointer(&blockFreeList{}))

	assert.Nil(t, freeList.First(base))

	freeList.Len = 1
	assert.NotNil(t, freeList.First(base))
}

func TestDataSizeToIndex(t *testing.T) {
	assert.Equal(t, 0, dataSizeToIndex(1))
	assert.Equal(t, FreeListLen, dataSizeToIndex(1<<FreeListLen))
}

func TestMaxSize(t *testing.T) {
	allocator := mockAllocator
	container := &lruAndFreeContainer{}
	container.Init(allocator.Base())
	assert.Equal(t, uint64(16*MB), container.MaxSize())
}
