package fastcache

import (
	"errors"
	"math"
	"unsafe"
)

var (
	ErrIndexOutOfRange = errors.New("index out of range")
	ErrLRUListEmpty    = errors.New("LRU list is empty")
)

const FreeListLen = 25 // 1 2 4 8 ... 16M, max free DataNode size = 16M

type lruAndFreeContainer struct {
	freeLists [FreeListLen]blockFreeList
	lruLists  [FreeListLen]list
}

func (b *lruAndFreeContainer) Init(base uintptr) {
	size := 1
	for i := 0; i < len(b.freeLists); i++ {
		freeList := &b.freeLists[i]
		freeList.Size = uint64(size)
		freeList.Index = uint8(i)

		lruList := &b.lruLists[i]
		lruList.Init(base)

		size *= 2
	}
}

func (b *lruAndFreeContainer) Get(dataSize uint64) (*blockFreeList, error) {
	if dataSize == 0 {
		return nil, errors.New("data size is zero")
	}

	idx := dataSizeToIndex(dataSize)
	if idx > len(b.freeLists)-1 {
		return nil, ErrIndexOutOfRange
	}

	return &b.freeLists[idx], nil
}

func (b *lruAndFreeContainer) GetIndex(idx uint8) *blockFreeList {
	return &b.freeLists[idx]
}

func (b *lruAndFreeContainer) MaxSize() uint64 {
	return b.freeLists[len(b.freeLists)-1].Size
}

func (b *lruAndFreeContainer) Len() int {
	return len(b.freeLists)
}

// PreAlloc 预先分配到free list里面
func (b *lruAndFreeContainer) PreAlloc(allocator Allocator, limitIndex int) error {
	for i := 0; i < limitIndex; i++ {
		freeList := &b.freeLists[i]
		fixedSize := freeList.Size
		nodeSize := uint64(sizeOfDataNode) + fixedSize
		allocSize := nodeSize
		ptr, offset, err := allocator.Alloc(allocSize)
		if err != nil {
			return err
		}
		node := (*DataNode)(ptr)
		node.Reset()
		node.FreeBlockIndex = freeList.Index
		head := freeList.First(allocator.Base())
		if head == nil {
			freeList.Head = offset
		} else {
			// 头插, 把当前的head, 前面插入node节点
			next := head
			node.Next = next.Offset(allocator.Base())
			head = node
		}
		freeList.Len++
	}
	return nil
}

func (b *lruAndFreeContainer) Alloc(allocator Allocator, dataSize uint64) (*DataNode, error) {
	freeList, err := b.Get(dataSize)
	if err != nil {
		return nil, err
	}

	// 一个节点需要的字节数等于链表头长度+定长数据长度
	fixedSize := freeList.Size
	nodeSize := uint64(sizeOfDataNode) + fixedSize

	if freeList.Len == 0 {
		// if alloc size less than PageSize will alloc PageSize, other alloc nodeSize
		allocSize := nodeSize
		if nodeSize < PageSize {
			allocSize = (uint64(PageSize)/nodeSize + 1) * nodeSize
		}
		_, offset, err := allocator.Alloc(allocSize)
		if err != nil {
			return nil, err
		}
		// 设置第一个链表节点的offset
		nodeLen := allocSize / nodeSize
		head := freeList.First(allocator.Base())
		// 头插法
		for i := 0; i < int(nodeLen); i++ {
			ptr := unsafe.Pointer(allocator.Base() + uintptr(offset))
			node := (*DataNode)(ptr)
			node.Reset()
			// 填写数据的指针位置
			node.FreeBlockIndex = freeList.Index
			if head == nil {
				head = node
			} else {
				// 头插, 把当前的head, 前面插入node节点
				next := head
				node.Next = next.Offset(allocator.Base())
				head = node
			}
			offset += nodeSize
		}
		freeList.Len += uint32(nodeLen)
		if head != nil {
			freeList.Head = head.Offset(allocator.Base())
		}
	}

	// 把第一个链表节点取出
	node := freeList.First(allocator.Base())
	freeList.Head = node.Next
	freeList.Len--
	// 断开与这个链表的关联, 变成一个独立的node
	node.Next = 0

	return node, nil
}

func (b *lruAndFreeContainer) MoveToFront(base uintptr, node *DataNode, lruNode *listNode) {
	lruList := &b.lruLists[node.FreeBlockIndex]
	lruList.MoveToFront(base, lruNode)
}

func (b *lruAndFreeContainer) PushFront(base uintptr, node *DataNode, lruNode *listNode) {
	lruList := &b.lruLists[node.FreeBlockIndex]
	lruList.PushFront(base, lruNode)
}

func (b *lruAndFreeContainer) Free(base uintptr, node *DataNode, lruNode *listNode) {
	// TODO 优化代码, 把freeList跟LRU区分更加合理
	// remove LRU list
	lruList := &b.lruLists[node.FreeBlockIndex]
	lruList.Remove(base, lruNode)

	// insert to free list
	freeList := b.GetIndex(node.FreeBlockIndex)
	node.Reset()
	next := freeList.Head
	node.Next = next
	freeList.Head = uint64(uintptr(unsafe.Pointer(node)) - base)
	freeList.Len++
}

func (b *lruAndFreeContainer) Evict(allocator Allocator, size uint64, onEvict func(node *listNode)) error {
	index := dataSizeToIndex(size)
	lruList := &b.lruLists[index]
	if lruList.Len() == 0 {
		return ErrLRUListEmpty
	}
	oldest := lruList.Back(allocator.Base())
	onEvict(oldest)
	lruList.remove(allocator.Base(), oldest)
	return nil
}

type blockFreeList struct {
	Head  uint64 // head of data DataNode
	Len   uint32 // data len
	Size  uint64 // block bytes size
	Index uint8
}

func (bl *blockFreeList) First(base uintptr) *DataNode {
	if bl.Len == 0 {
		return nil
	}
	return (*DataNode)(unsafe.Pointer(base + uintptr(bl.Head)))
}

func dataSizeToIndex(size uint64) int {
	v := math.Log2(float64(size))
	return int(math.Ceil(v))
}
