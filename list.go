package memlru

import "unsafe"

type list struct {
	Len    uint32
	Offset uint64 // listElement linkedNode offset
}

func (l *list) Range(memMgr *MemoryManager, f func(el *listElement) bool) {
	if l.Len == 0 {
		return
	}
	base := memMgr.basePtr()
	RangeNode(base, l.Offset, func(node *LinkedNode) bool {
		el := NodeConvertTo[listElement](base, node)
		if !f(el) {
			return false
		}
		return true
	})
}

func (l *list) Set(memMgr *MemoryManager, key string, value []byte) error {
	_, findNode, findEl := l.findNode(memMgr, key)
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
			var node *LinkedNode
			node, err := memMgr.allocOne(uint64(len(value)))
			if err != nil {
				return err
			}
			valNode = node
			// 重新绑定element的val data node
			findEl.ValOffset = valNode.Offset(memMgr.basePtr())
		}
		valNode.UpdateData(memMgr.basePtr(), value)
		return nil
	}
	// else not find key in list, need alloc new node
	// 申请一个数据块
	listElNode, err := memMgr.allocOne(uint64(sizeOfListElement))
	if err != nil {
		return err
	}
	listEl := (*listElement)(unsafe.Pointer(listElNode.DataPtr(memMgr.basePtr())))
	if err = listEl.Set(memMgr, key, value); err != nil {
		return err
	}
	// 更新list链表, 头插法
	next := l.Offset
	// 把item的头指针指向当前的listElNode
	l.Offset = listElNode.Offset(memMgr.basePtr())
	// 更新next
	listElNode.Next = next
	// hashed array len + 1
	l.Len++
	return nil
}

func (l *list) Del(memMgr *MemoryManager, key string) error {
	prevNode, findNode, findEl := l.findNode(memMgr, key)

	// not found
	if findNode == nil {
		return ErrNotFound
	}

	if prevNode == nil {
		// 就说明这个是头节点, 需要更新list的头节点指向
		l.Offset = findNode.Next
	} else {
		prevNode.Next = findNode.Next
	}

	// free listElement key and value data
	findEl.Free(memMgr)
	// free list element node
	memMgr.free(findNode)
	l.Len--
	// list中没有任何element, 就把head offset = 0
	if l.Len == 0 {
		l.Offset = 0
	}
	return nil
}

func (l *list) Find(memMgr *MemoryManager, key string) *listElement {
	_, _, find := l.findNode(memMgr, key)
	return find
}

func (l *list) Reset() {
	l.Len = 0
	l.Offset = 0
}

func (l *list) findNode(memMgr *MemoryManager, key string) (prevNode *LinkedNode, findNode *LinkedNode, findEl *listElement) {
	RangeNode(memMgr.basePtr(), l.Offset, func(node *LinkedNode) bool {
		el := NodeConvertTo[listElement](memMgr.basePtr(), node)
		if el.ToKey(memMgr) == key {
			findNode = node
			findEl = el
			return false
		}
		prevNode = node
		return true
	})
	return
}

// listElement 是一个keyNode + valNode组成
type listElement struct {
	KeyOffset uint64 // key node offset
	ValOffset uint64 // val node offset
}

func (l *listElement) Set(memMgr *MemoryManager, key string, value []byte) error {
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

	l.KeyOffset = keyNode.Offset(memMgr.basePtr())
	l.ValOffset = valNode.Offset(memMgr.basePtr())
	return nil
}

func (l *listElement) ToKey(memMgr *MemoryManager) string {
	node := memMgr.toLinkedNode(l.KeyOffset)
	return string(node.Data(memMgr.basePtr()))
}

func (l *listElement) Free(memMgr *MemoryManager) {
	l.FreeKey(memMgr)
	l.FreeValue(memMgr)
}

func (l *listElement) FreeKey(memMgr *MemoryManager) {
	keyNode := memMgr.toLinkedNode(l.KeyOffset)
	memMgr.free(keyNode)
}

func (l *listElement) FreeValue(memMgr *MemoryManager) {
	valNode := memMgr.toLinkedNode(l.ValOffset)
	memMgr.free(valNode)
}

func (l *listElement) ValNode(memMgr *MemoryManager) *LinkedNode {
	return memMgr.toLinkedNode(l.ValOffset)
}
