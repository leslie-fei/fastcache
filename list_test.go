package fastcache

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

func TestListNode_SetPrev(t *testing.T) {
	base := mockAllocator.Base()
	node := &listNode{}
	prevNode := &listNode{}

	node.SetPrev(base, prevNode)
	assert.Equal(t, uint64(uintptr(unsafe.Pointer(prevNode))-base), node.prev)
}

func TestListNode_SetNext(t *testing.T) {
	base := mockAllocator.Base()
	node := &listNode{}
	nextNode := &listNode{}

	node.SetNext(base, nextNode)
	assert.Equal(t, uint64(uintptr(unsafe.Pointer(nextNode))-base), node.next)
}

func TestListNode_Prev(t *testing.T) {
	base := mockAllocator.Base()
	node := &listNode{}
	prevNode := &listNode{}
	node.SetPrev(base, prevNode)

	assert.Equal(t, prevNode, node.Prev(base))
}

func TestListNode_Next(t *testing.T) {
	base := mockAllocator.Base()
	node := &listNode{}
	nextNode := &listNode{}
	node.SetNext(base, nextNode)

	assert.Equal(t, nextNode, node.Next(base))
}

func TestListNode_Offset(t *testing.T) {
	base := mockAllocator.Base()
	node := &listNode{}

	assert.Equal(t, uint64(uintptr(unsafe.Pointer(node))-base), node.Offset(base))
}

func TestList_Init(t *testing.T) {
	l := &list{}
	base := mockAllocator.Base()
	l.Init(base)

	assert.Equal(t, &l.root, l.root.Next(base))
	assert.Equal(t, &l.root, l.root.Prev(base))
	assert.Equal(t, uint64(0), l.len)
}

func TestList_Len(t *testing.T) {
	l := &list{}
	l.len = 10
	assert.Equal(t, uint64(10), l.Len())
}

func TestList_Front(t *testing.T) {
	l := &list{}
	base := mockAllocator.Base()
	l.Init(base)

	assert.Nil(t, l.Front(base))

	node := &listNode{}
	l.PushFront(base, node)
	assert.Equal(t, node, l.Front(base))
}

func TestList_Back(t *testing.T) {
	l := &list{}
	base := mockAllocator.Base()
	l.Init(base)

	assert.Nil(t, l.Back(base))

	node := &listNode{}
	l.PushBack(base, node)
	assert.Equal(t, node, l.Back(base))
}

func TestList_Remove(t *testing.T) {
	l := &list{}
	base := mockAllocator.Base()
	l.Init(base)

	node := &listNode{}
	l.PushFront(base, node)
	l.Remove(base, node)

	assert.Nil(t, l.Front(base))
	assert.Equal(t, uint64(0), l.len)
}

func TestList_PushFront(t *testing.T) {
	l := &list{}
	base := mockAllocator.Base()
	l.Init(base)

	node := &listNode{}
	l.PushFront(base, node)

	assert.Equal(t, node, l.Front(base))
	assert.Equal(t, uint64(1), l.len)
}

func TestList_PushBack(t *testing.T) {
	l := &list{}
	base := mockAllocator.Base()
	l.Init(base)

	node := &listNode{}
	l.PushBack(base, node)

	assert.Equal(t, node, l.Back(base))
	assert.Equal(t, uint64(1), l.len)
}

func TestList_MoveToFront(t *testing.T) {
	l := &list{}
	base := mockAllocator.Base()
	l.Init(base)

	nodes := make([]listNode, 10)
	for i := 0; i < len(nodes); i++ {
		node := &nodes[i]
		l.PushBack(base, node)
	}
	node := &nodes[5]
	l.MoveToFront(base, node)

	assert.Equal(t, node, l.Front(base))
}

func TestList_MoveToBack(t *testing.T) {
	l := &list{}
	base := mockAllocator.Base()
	l.Init(base)

	nodes := make([]listNode, 10)
	for i := 0; i < len(nodes); i++ {
		node := &nodes[i]
		l.PushBack(base, node)
	}
	node := &nodes[5]
	l.MoveToBack(base, node)
	l.MoveToBack(base, node)
	l.move(base, l.Back(base), node)
	assert.Equal(t, node, l.Back(base))
}

func TestList_Insert(t *testing.T) {
	l := &list{}
	base := mockAllocator.Base()
	l.Init(base)

	node := &listNode{}
	l.insert(base, node, &l.root)

	assert.Equal(t, node, l.Front(base))
	assert.Equal(t, uint64(1), l.len)
}

func TestList_RemoveInternal(t *testing.T) {
	l := &list{}
	base := mockAllocator.Base()
	l.Init(base)

	node := &listNode{}
	l.PushFront(base, node)
	l.remove(base, node)

	assert.Nil(t, l.Front(base))
	assert.Equal(t, uint64(0), l.len)
}

func TestList_Move(t *testing.T) {
	l := &list{}
	base := mockAllocator.Base()
	l.Init(base)

	node := &listNode{}
	l.PushFront(base, node)

	otherNode := &listNode{}
	l.PushBack(base, otherNode)
	l.move(base, node, &l.root)

	assert.Equal(t, node, l.Front(base))
	assert.Equal(t, otherNode, l.Back(base))
}
