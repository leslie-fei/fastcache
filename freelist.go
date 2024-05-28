package fastcache

import (
	"math"
	"unsafe"
)

var sizeOfFreeStore = unsafe.Sizeof(freeStore{})

type freeStore struct {
	freeLists [25]freeList
}

func (f *freeStore) init(all *allocator) error {
	for i := 0; i < len(f.freeLists); i++ {
		fl := &f.freeLists[i]
		fl.reset()
		fl.index = uint8(i)
		fl.size = 1 << i
		// 小于1KB的数据, 预分配
		if fl.size <= 16*KB {
			if err := fl.alloc(all, 10); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *freeStore) getIndex(idx uint8) *freeList {
	return &f.freeLists[idx]
}

func (f *freeStore) get(all *allocator, size uint32) (*dataNode, error) {
	index := sizeToIndex(size)
	if int(index) >= len(f.freeLists) {
		return nil, ErrIndexOutOfRange
	}
	fl := &f.freeLists[index]
	if fl.len == 0 {
		// 没有可用的free node, 需要去内存申请
		// freeList链表中的是一个dataNodeHead + [size]byte
		var allLen uint32 = 1
		if size < KB {
			allLen = 1024
		} else if size < MB {
			allLen = 10
		}
		if err := fl.alloc(all, allLen); err != nil {
			return nil, err
		}
	}

	if fl.len == 0 {
		return nil, ErrFreeListIsEmpty
	}

	node := fl.first(all)
	fl.firstDataNodeOffset = node.next
	fl.len--
	node.next = 0
	return node, nil
}

func (f *freeStore) free(all *allocator, node *dataNode) {
	fl := &f.freeLists[node.freeIndex]
	node.reset()
	first := fl.firstDataNodeOffset
	node.next = first
	node.freeIndex = fl.index
	fl.firstDataNodeOffset = uint64(uintptr(unsafe.Pointer(node)) - all.base())
	fl.len++
}

func sizeToIndex(size uint32) uint8 {
	v := math.Log2(float64(size))
	return uint8(math.Ceil(v))
}

type freeList struct {
	index               uint8
	len                 uint32
	size                uint32
	firstDataNodeOffset uint64
}

func (f *freeList) reset() {
	*f = freeList{}
}

func (f *freeList) first(all *allocator) *dataNode {
	if f.len == 0 {
		return nil
	}
	return toDataNode(all, f.firstDataNodeOffset)
}

func (f *freeList) alloc(all *allocator, length uint32) error {
	nodeSize := uint64(sizeOfDataNode) + uint64(f.size)
	total := nodeSize * uint64(length)
	_, offset, err := all.alloc(total)
	if err != nil {
		return err
	}
	head := f.first(all)
	// 头插法
	for i := 0; i < int(length); i++ {
		node := toDataNode(all, offset)
		node.reset()
		// 填写数据的指针位置
		node.freeIndex = f.index
		if head == nil {
			head = node
		} else {
			// 头插, 把当前的head, 前面插入node节点
			first := head
			node.next = first.offset(all)
			head = node
		}
		offset += nodeSize
		f.len++
	}

	if head != nil {
		f.firstDataNodeOffset = head.offset(all)
	}

	return nil
}
