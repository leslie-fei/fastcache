package memlru

import (
	"errors"
	"reflect"
	"unsafe"
)

var (
	ErrNotFound      = errors.New("key not found")
	ErrValueTooLarge = errors.New("value too large")
	ErrKeyTooLarge   = errors.New("key too large")
)

// HashMap fixed size HashMap
type HashMap struct {
	Len   uint32
	array [1024]list
}

func (m *HashMap) Get(memMgr *MemoryManager, key string) ([]byte, error) {
	el, err := m.get(memMgr, key)
	if err != nil {
		return nil, err
	}
	node := (*LinkedNode)(memMgr.offset(el.ValOffset))
	return node.Data(memMgr.basePtr()), nil
}

func (m *HashMap) Set(memMgr *MemoryManager, key string, value []byte) error {
	item := m.item(key)

	found := false
	var err error
	item.Range(memMgr, func(el *listElement) bool {
		if el.ToKey() == key {
			// found
			valNode := memMgr.toLinkedNode(el.ValOffset)
			// 判断是否超过原来的节点最大可以容纳的size
			if len(value) > DataSizeMap[FreeDataType(valNode.FreeType)] {
				// 如果超过需要重新分配一个node来装数据
				oldNode := valNode
				var node *LinkedNode
				node, err = memMgr.allocOne(uint64(len(value)))
				if err != nil {
					return false
				}
				// 释放老节点到freeList
				memMgr.free(oldNode)
				valNode = node
				// 重新绑定element的val data node
				el.ValOffset = valNode.Offset(memMgr.basePtr())
			}
			valNode.UpdateData(memMgr.basePtr(), value)
			found = true
			// break foreach
			return false
		}
		return true
	})

	if err != nil {
		return err
	}

	if found {
		return nil
	}

	// not found should to alloc one data node
	// 申请一个数据块
	listElNode, err := memMgr.allocOne(uint64(sizeOfListElement))
	if err != nil {
		return err
	}
	listEl := (*listElement)(unsafe.Pointer(listElNode.DataPtr(memMgr.basePtr())))

	// set key
	ss := (*reflect.StringHeader)(unsafe.Pointer(&key))
	memmove(unsafe.Pointer(&listEl.Key), unsafe.Pointer(ss.Data), uintptr(ss.Len))
	listEl.KeyLen = uint16(ss.Len)

	// set value
	// alloc data node to set value
	valNode, err := memMgr.allocOne(uint64(len(value)))
	if err != nil {
		return err
	}
	valNode.UpdateData(memMgr.basePtr(), value)
	// 然后把这个链表的指针offset写入到element中
	listEl.ValOffset = valNode.Offset(memMgr.basePtr())

	// 更新list链表, 头插法
	next := item.Offset
	// 把item的头指针指向当前的listElNode
	item.Offset = listElNode.Offset(memMgr.basePtr())
	// 更新next
	listElNode.Next = next
	// hashed array len + 1
	item.Len++
	// hashmap total len + 1
	m.Len++

	return nil
}

func (m *HashMap) Del(memMgr *MemoryManager, key string) error {
	item := m.item(key)
	var findNode *LinkedNode
	// 头节点的偏移量
	offset := item.Offset
	var prev uint64
	for i := 0; i < int(item.Len); i++ {
		elNode := memMgr.toLinkedNode(offset)
		elPtr := memMgr.offset(elNode.DataOffset)
		el := (*listElement)(elPtr)
		if el.ToKey() == key {
			findNode = elNode
			break
		}
		prev = offset
		offset = elNode.Next
	}

	// not found
	if findNode == nil {
		return ErrNotFound
	}

	if prev == 0 {
		// 就说明这个是头节点, 需要更新list的头节点指向
		item.Offset = findNode.Next
	} else {
		prevNode := memMgr.toLinkedNode(prev)
		prevNode.Next = findNode.Next
	}
	memMgr.free(findNode)
	item.Len--
	m.Len--
	// list中没有任何element, 就把head offset = 0
	if item.Len == 0 {
		item.Offset = 0
	}
	return nil
}

func (m *HashMap) item(key string) *list {
	hash := xxHashString(key)
	index := uint64(hash) % uint64(len(m.array))
	item := &m.array[index]
	return item
}

func (m *HashMap) get(memMgr *MemoryManager, key string) (*listElement, error) {
	item := m.item(key)
	find := item.Find(memMgr, key)
	if find != nil {
		return find, nil
	}
	return nil, ErrNotFound
}

type list struct {
	Len    uint32
	Offset uint64 // linkNode ptr offset
}

func (l *list) Range(memMgr *MemoryManager, f func(el *listElement) bool) {
	if l.Len == 0 {
		return
	}
	for node := memMgr.toLinkedNode(l.Offset); node != nil; node = memMgr.toLinkedNode(node.Next) {
		el := (*listElement)(memMgr.offset(node.DataOffset))
		if !f(el) {
			return
		}
	}
}

func (l *list) Find(memMgr *MemoryManager, key string) *listElement {
	var find *listElement
	l.Range(memMgr, func(el *listElement) bool {
		if el.ToKey() == key {
			find = el
			return false
		}
		return true
	})
	return find
}

func (l *list) Reset() {
	l.Len = 0
	l.Offset = 0
}

type listElement struct {
	KeyLen    uint16
	Key       [32]byte
	ValOffset uint64 // basePtr offset
}

func (l *listElement) ToKey() string {
	return string(l.Key[:l.KeyLen])
}
