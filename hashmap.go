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
	memMgr *MemoryManager
	Len    uint32
	array  [1024]list
}

func (m *HashMap) Get(key string) ([]byte, error) {
	el, err := m.get(key)
	if err != nil {
		return nil, err
	}
	node := (*LinkedNode)(m.memMgr.Offset(el.ValOffset))
	return node.Data(m.memMgr.BasePtr()), nil
}

func (m *HashMap) Set(key string, value []byte) error {
	item := m.item(key)

	found := false
	var err error
	item.Range(m.memMgr, func(el *listElement) bool {
		if el.ToKey() == key {
			// found
			valNode := m.memMgr.ToLinkedNode(el.ValOffset)
			// 判断是否超过原来的节点最大可以容纳的size
			if len(value) > DataSizeMap[FreeDataType(valNode.FreeType)] {
				// 如果超过需要重新分配一个node来装数据
				oldNode := valNode
				var node *LinkedNode
				node, err = m.memMgr.allocOne(uint64(len(value)))
				if err != nil {
					return false
				}
				// 释放老节点到freeList
				m.memMgr.free(oldNode)
				valNode = node
				// 重新绑定element的val data node
				el.ValOffset = valNode.Offset(m.memMgr.BasePtr())
			}
			valNode.UpdateData(m.memMgr.BasePtr(), value)
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
	listElNode, err := m.memMgr.allocOne(uint64(sizeOfListElement))
	if err != nil {
		return err
	}
	listEl := (*listElement)(unsafe.Pointer(listElNode.DataPtr(m.memMgr.BasePtr())))

	// set key
	ss := (*reflect.StringHeader)(unsafe.Pointer(&key))
	memmove(unsafe.Pointer(&listEl.Key), unsafe.Pointer(ss.Data), uintptr(ss.Len))
	listEl.KeyLen = uint16(ss.Len)

	// set value
	// alloc data node to set value
	valNode, err := m.memMgr.allocOne(uint64(len(value)))
	if err != nil {
		return err
	}
	valNode.UpdateData(m.memMgr.BasePtr(), value)
	// 然后把这个链表的指针offset写入到element中
	listEl.ValOffset = valNode.Offset(m.memMgr.BasePtr())

	// 更新item链表, 头插法
	next := item.Offset
	// 把item的头指针指向当前的listElNode
	item.Offset = listElNode.Offset(m.memMgr.BasePtr())
	// 更新next
	listElNode.Next = next
	// hashed array len + 1
	item.Len++
	// hashmap total len + 1
	m.Len++

	return nil
}

func (m *HashMap) Del(key string) {
	item := m.item(key)
	memMgr := m.memMgr
	var findNode *LinkedNode
	// 头节点的偏移量
	offset := item.Offset
	var prev uint64
	for i := 0; i < int(item.Len); i++ {
		elNode := memMgr.ToLinkedNode(offset)
		el := (*listElement)(memMgr.Offset(elNode.DataOffset))
		if el.ToKey() == key {
			findNode = elNode
			break
		}
		prev = offset
		offset = elNode.Next
	}

	// not found
	if findNode == nil {
		return
	}

	if prev == 0 {
		// 就说明这个是头节点, 需要更新list的头节点指向
		item.Offset = findNode.Next
	} else {
		prevNode := memMgr.ToLinkedNode(prev)
		prevNode.Next = findNode.Next
	}
	memMgr.free(findNode)
	item.Len--
	m.Len--
}

func (m *HashMap) item(key string) *list {
	hash := xxHashString(key)
	index := uint64(hash) % uint64(len(m.array))
	item := &m.array[index]
	return item
}

func (m *HashMap) get(key string) (*listElement, error) {
	item := m.item(key)
	find := item.Find(m.memMgr, key)
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
	for node := memMgr.ToLinkedNode(l.Offset); node != nil; node = memMgr.ToLinkedNode(node.Next) {
		el := (*listElement)(memMgr.Offset(node.DataOffset))
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

func (l *listElement) FreeTo(memMgr *MemoryManager) {

}
