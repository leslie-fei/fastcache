package memlru

import (
	"errors"
	"reflect"
	"unsafe"
)

// HashMap fixed size HashMap
type HashMap struct {
	memMgr *MemoryManager
	Len    uint32
	array  [1]list
}

func (m *HashMap) Get(key string) []byte {
	hash := xxHashString(key)
	index := uint64(hash) % uint64(len(m.array))
	item := m.array[index]
	els := item.Elements(m.memMgr.Offset(item.Offset))
	for i := 0; i < len(els); i++ {
		el := els[i]
		if el.ValLen > 0 && el.ToKey() == key {
			node := (*LinkedNode16)(m.memMgr.Offset(el.ValOffset))
			return node.Data[:el.ValLen]
		}
	}
	return nil
}

func (m *HashMap) Set(key string, value []byte) error {
	hash := xxHashString(key)
	index := uint64(hash) % uint64(len(m.array))
	item := &m.array[index]
	if item.Len == 0 {
		item.grow(m.memMgr)
	}
	// TODO 优化查询, 因为值不存在的时候如果通过遍历需要O(n)
	elements := item.Elements(m.memMgr.Offset(item.Offset))
	var free *listElement
	for i := 0; i < len(elements); i++ {
		el := &elements[i]
		if el.ValLen > 0 && el.ToKey() == key {
			// found
			el.ValLen = int32(len(value))
			node := (*LinkedNode16)(m.memMgr.Offset(el.ValOffset))
			// value把数据移到链表节点上
			bh := (*reflect.SliceHeader)(unsafe.Pointer(&value))
			memmove(unsafe.Pointer(&node.Data), unsafe.Pointer(bh.Data), uintptr(el.ValLen))
			return nil
		}

		if free == nil && el.ValLen == 0 {
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
	free.KeyLen = int32(ss.Len)
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
	KeyLen    int32
	Key       [32]byte
	ValLen    int32
	ValOffset uint64
}

func (l *listElement) ToKey() string {
	return string(l.Key[:l.KeyLen])
}

/*type String struct {
	Offset uint64
	Len    uint32
}

func (str String) String(base unsafe.Pointer) string {
	s := ""
	ss := (*reflect.StringHeader)(unsafe.Pointer(&s))
	ss.Data = uintptr(base) + uintptr(str.Offset)
	ss.Len = int(str.Len)
	return s
}

func ToKey(s string, offset uint64) String {
	var str = String{
		Offset: offset,
		Len:    uint32(len(s)),
	}
	return str
}
*/
