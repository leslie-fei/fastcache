package fastcache

import (
	"reflect"
	"unsafe"
)

type hashMapBucket struct {
	len    uint32
	offset uint64 // hashMapBucketElement linkedNode offset
}

func (l *hashMapBucket) Add(base uintptr, node *DataNode) {
	// 更新list链表, 头插法
	next := l.offset
	// 把item的头指针指向当前的listElNode
	l.offset = node.Offset(base)
	// 更新next
	node.Next = next
	// hashed array len + 1
	l.len++
}

func (l *hashMapBucket) Del(prevNode *DataNode, findNode *DataNode) error {
	// not found
	if findNode == nil {
		return ErrNotFound
	}

	if prevNode == nil {
		// 就说明这个是头节点, 需要更新list的头节点指向
		l.offset = findNode.Next
	} else {
		prevNode.Next = findNode.Next
	}

	l.len--
	// list中没有任何element, 就把head offset = 0
	if l.len == 0 {
		l.offset = 0
	}

	return nil
}

func (l *hashMapBucket) Reset() {
	l.len = 0
	l.offset = 0
}

func (l *hashMapBucket) FindNode(base uintptr, key string) (prevNode *DataNode, findNode *DataNode, findEl *hashMapBucketElement) {
	if l.len == 0 {
		return nil, nil, nil
	}

	offset := l.offset
	for i := 0; i < int(l.len); i++ {
		node := ToDataNode(base, offset)
		el := NodeTo[hashMapBucketElement](node)
		if el.Equals(key) {
			findNode = node
			findEl = el
			return
		}
		prevNode = node
		offset = node.Next
	}

	return
}

// hashMapBucketElement key DataNode string + val DataNode []byte
type hashMapBucketElement struct {
	lruNode listNode
	keyLen  uint32 // key length
	valLen  uint32 // val length
}

func (l *hashMapBucketElement) Equals(key string) bool {
	if l.keyLen != uint32(len(key)) {
		return false
	}
	keyPtr := uintptr(unsafe.Pointer(l)) + sizeOfHashmapBucketElement
	kh := (*reflect.StringHeader)(unsafe.Pointer(&key))
	return memequal(unsafe.Pointer(keyPtr), unsafe.Pointer(kh.Data), uintptr(len(key)))
}

func (l *hashMapBucketElement) Key() string {
	var s string
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	keyPtr := uintptr(unsafe.Pointer(l)) + sizeOfHashmapBucketElement
	sh.Data = keyPtr
	sh.Len = int(l.keyLen)
	return s
}

func (l *hashMapBucketElement) UpdateValue(value []byte) {
	valPtr := uintptr(unsafe.Pointer(l)) + sizeOfHashmapBucketElement + uintptr(l.keyLen)
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&value))
	memmove(unsafe.Pointer(valPtr), unsafe.Pointer(bh.Data), uintptr(len(value)))
	l.valLen = uint32(len(value))
}

func (l *hashMapBucketElement) Value() []byte {
	valPtr := uintptr(unsafe.Pointer(l)) + sizeOfHashmapBucketElement + uintptr(l.keyLen)
	var ss []byte
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&ss))
	sh.Data = valPtr
	sh.Len = int(l.valLen)
	sh.Cap = sh.Len
	return ss
}

func hashmapElementSize(key string, value []byte) uint64 {
	return uint64(sizeOfHashmapBucketElement) + uint64(len(key)) + uint64(len(value))
}
