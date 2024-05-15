package memlru

import (
	"reflect"
	"unsafe"
)

type hashmapSlot struct {
	len    uint32
	offset uint64 // HashmapSlotElement linkedNode offset
}

func (l *hashmapSlot) Range(memMgr *MemoryManager, f func(el *HashmapSlotElement) bool) {
	if l.len == 0 {
		return
	}
	base := memMgr.basePtr()
	RangeNode(base, l.offset, func(node *DataNode) bool {
		el := NodeConvertTo[HashmapSlotElement](base, node)
		if !f(el) {
			return false
		}
		return true
	})
}

func (l *hashmapSlot) Set(memMgr *MemoryManager, key string, value []byte) (exists bool, node *DataNode, err error) {
	_, findNode, findEl := l.FindNode(memMgr, key)
	if findNode != nil {
		// found
		valNode := findEl.ValNode(memMgr)
		blockSize := memMgr.blockSize(valNode)
		// 判断是否超过原来的节点最大可以容纳的size
		if len(value) > int(blockSize) {
			// 如果超过需要重新分配一个node来装数据
			// 释放老节点到freeList
			findEl.FreeValue(memMgr)
			// 申请新的数据节点
			node, err = memMgr.allocOne(uint64(len(value)))
			if err != nil {
				return
			}
			valNode = node
			// 重新绑定element的val data node
			findEl.valOffset = valNode.Offset(memMgr.basePtr())
		}
		valNode.UpdateData(memMgr.basePtr(), value)
		return true, node, nil
	}
	// else not find key in hashmapSlot, need alloc new node
	// 申请一个数据块
	node, err = memMgr.allocOne(uint64(sizeOfListElement))
	if err != nil {
		return
	}
	el := (*HashmapSlotElement)(unsafe.Pointer(node.DataPtr(memMgr.basePtr())))
	if err = el.Set(memMgr, key, value); err != nil {
		return
	}
	// 更新list链表, 头插法
	next := l.offset
	// 把item的头指针指向当前的listElNode
	l.offset = node.Offset(memMgr.basePtr())
	// 更新next
	node.Next = next
	// hashed array len + 1
	l.len++
	return false, node, nil
}

func (l *hashmapSlot) Del(memMgr *MemoryManager, key string) (el *HashmapSlotElement, err error) {
	prevNode, findNode, findEl := l.FindNode(memMgr, key)

	// not found
	if findNode == nil {
		return nil, ErrNotFound
	}

	if prevNode == nil {
		// 就说明这个是头节点, 需要更新list的头节点指向
		l.offset = findNode.Next
	} else {
		prevNode.Next = findNode.Next
	}

	// free HashmapSlotElement key and value data
	findEl.Free(memMgr)
	// free hashmapSlot element node
	memMgr.free(findNode)
	l.len--
	// list中没有任何element, 就把head offset = 0
	if l.len == 0 {
		l.offset = 0
	}
	return findEl, nil
}

func (l *hashmapSlot) Find(memMgr *MemoryManager, key string) *HashmapSlotElement {
	_, _, find := l.FindNode(memMgr, key)
	return find
}

func (l *hashmapSlot) Reset() {
	l.len = 0
	l.offset = 0
}

func (l *hashmapSlot) FindNode(memMgr *MemoryManager, key string) (prevNode *DataNode, findNode *DataNode, findEl *HashmapSlotElement) {
	RangeNode(memMgr.basePtr(), l.offset, func(node *DataNode) bool {
		el := NodeConvertTo[HashmapSlotElement](memMgr.basePtr(), node)
		if el.Equals(memMgr, key) {
			findNode = node
			findEl = el
			return false
		}
		prevNode = node
		return true
	})
	return
}

// HashmapSlotElement 是一个keyNode + valNode组成
type HashmapSlotElement struct {
	lruListNodeOffset uint64 // hashmap element in lru list
	keyOffset         uint64 // key node offset
	valOffset         uint64 // val node offset
}

func (l *HashmapSlotElement) Set(memMgr *MemoryManager, key string, value []byte) error {
	// set key
	keyNode, err := memMgr.allocOne(uint64(len(key)))
	if err != nil {
		return err
	}

	keyNode.UpdateString(memMgr.basePtr(), key)

	// set value
	// alloc data node to set value
	valNode, err := memMgr.allocOne(uint64(len(value)))
	if err != nil {
		return err
	}
	valNode.UpdateData(memMgr.basePtr(), value)

	l.keyOffset = keyNode.Offset(memMgr.basePtr())
	l.valOffset = valNode.Offset(memMgr.basePtr())
	return nil
}

func (l *HashmapSlotElement) ToKey(memMgr *MemoryManager) string {
	node := memMgr.toLinkedNode(l.keyOffset)
	return string(node.Data(memMgr.basePtr()))
}

func (l *HashmapSlotElement) Equals(memMgr *MemoryManager, key string) bool {
	node := memMgr.toLinkedNode(l.keyOffset)
	if len(key) != int(node.Len) {
		return false
	}
	ptr := node.DataPtr(memMgr.basePtr())
	kh := (*reflect.StringHeader)(unsafe.Pointer(&key))
	return memequal(unsafe.Pointer(ptr), unsafe.Pointer(kh.Data), uintptr(len(key)))
}

func (l *HashmapSlotElement) Free(memMgr *MemoryManager) {
	l.FreeKey(memMgr)
	l.FreeValue(memMgr)
}

func (l *HashmapSlotElement) FreeKey(memMgr *MemoryManager) {
	keyNode := memMgr.toLinkedNode(l.keyOffset)
	memMgr.free(keyNode)
}

func (l *HashmapSlotElement) FreeValue(memMgr *MemoryManager) {
	valNode := memMgr.toLinkedNode(l.valOffset)
	memMgr.free(valNode)
}

func (l *HashmapSlotElement) ValNode(memMgr *MemoryManager) *DataNode {
	return memMgr.toLinkedNode(l.valOffset)
}
