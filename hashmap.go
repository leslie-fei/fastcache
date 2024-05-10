package memlru

import (
	"errors"
	"reflect"
	"unsafe"
)

var ErrNotFound = errors.New("key not found")

// HashMap fixed size HashMap
type HashMap struct {
	memMgr *MemoryManager
	Len    uint32
	array  [1]list
}

func (m *HashMap) Get(key string) ([]byte, error) {
	el, err := m.get(key)
	if err != nil {
		return nil, err
	}
	node := (*LinkedNode16)(m.memMgr.Offset(el.ValOffset))
	return node.Data[:el.ValLen], nil
}

func (m *HashMap) Set(key string, value []byte) error {
	item := m.item(key)
	if item.Len == 0 {
		item.grow(m.memMgr)
	}
	// TODO 优化查询, 因为值不存在的时候如果通过遍历需要O(n)
	elements := item.Elements(m.memMgr.Offset(item.Offset))
	var free *listElement
	for i := 0; i < len(elements); i++ {
		el := &elements[i]
		if !el.Deleted() && el.ToKey() == key {
			// found
			el.ValLen = int32(len(value))
			node := (*LinkedNode16)(m.memMgr.Offset(el.ValOffset))
			// value把数据移到链表节点上
			bh := (*reflect.SliceHeader)(unsafe.Pointer(&value))
			memmove(unsafe.Pointer(&node.Data), unsafe.Pointer(bh.Data), uintptr(el.ValLen))
			return nil
		}

		if free == nil && el.Deleted() {
			free = el
		}
	}

	if free == nil {
		// TODO grow
		// no free
		return errors.New("hashmap list elements no free")
	}

	// if not found but have free element
	// 设置key值
	ss := (*reflect.StringHeader)(unsafe.Pointer(&key))
	memmove(unsafe.Pointer(&free.Key), unsafe.Pointer(ss.Data), uintptr(ss.Len))
	free.KeyLen = uint16(ss.Len)
	// 设置value值
	free.ValLen = int32(len(value))
	// 申请一个数据块
	node, err := m.memMgr.alloc16()
	if err != nil {
		return err
	}
	// value把数据移到链表节点上
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&value))
	memmove(unsafe.Pointer(&node.Data), unsafe.Pointer(bh.Data), uintptr(free.ValLen))
	// 然后把这个链表的指针offset写入到element中
	free.ValOffset = node.Offset

	return nil
}

func (m *HashMap) Del(key string) error {
	el, err := m.get(key)
	if err != nil {
		return err
	}
	// 把val归还到free block list中
	node := (*LinkedNode16)(m.memMgr.Offset(el.ValOffset))
	m.memMgr.free16(node)
	el.MarkDeleted()
	return nil
}

func (m *HashMap) item(key string) *list {
	hash := xxHashString(key)
	index := uint64(hash) % uint64(len(m.array))
	item := &m.array[index]
	return item
}

func (m *HashMap) get(key string) (*listElement, error) {
	item := m.item(key)
	els := item.Elements(m.memMgr.Offset(item.Offset))
	for i := 0; i < len(els); i++ {
		el := &els[i]
		if !el.Deleted() && el.ToKey() == key {
			return el, nil
		}
	}
	return nil, ErrNotFound
}

type list struct {
	Len    uint32
	Offset uint64
}

func (l *list) grow(memMgr *MemoryManager) {
	/*size := l.Len * 2
	if size == 0 {
		size = 8
	}*/
	size := uint32(8)
	dataSize := uintptr(size) * sizeOfListElement
	_, headerOffset := memMgr.alloc(uint64(dataSize))
	l.Offset = headerOffset
	l.Len = size
	els := l.Elements(memMgr.Offset(headerOffset))
	for i := 0; i < len(els); i++ {
		els[i].Reset()
	}
}

func (l *list) Elements(ptr unsafe.Pointer) []listElement {
	var els []listElement
	elh := (*reflect.SliceHeader)(unsafe.Pointer(&els))
	elh.Cap = int(l.Len)
	elh.Len = elh.Cap
	elh.Data = uintptr(ptr)
	return els
}

type listElement struct {
	KeyLen    uint16
	Key       [32]byte
	ValLen    int32
	ValOffset uint64 // basePtr offset
}

func (l *listElement) Reset() {
	l.KeyLen = 0
	l.ValLen = -1
}

func (l *listElement) Deleted() bool {
	return l.ValLen == -1
}

func (l *listElement) MarkDeleted() {
	l.ValLen = -1
}

func (l *listElement) ToKey() string {
	return string(l.Key[:l.KeyLen])
}
