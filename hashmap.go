package fastcache

import (
	"reflect"
	"unsafe"
)

var (
	sizeOfHashmap              = unsafe.Sizeof(hashmap{})
	sizeOfHashmapBucket        = unsafe.Sizeof(hashmapBucket{})
	sizeOfHashmapBucketElement = unsafe.Sizeof(hashmapBucketElement{})
)

type hashmap struct {
	len           uint32
	bucketLen     uint32
	bucketsOffset uint64
}

func (m *hashmap) init(all *allocator, bucketLen uint32) error {
	m.len = 0
	m.bucketLen = bucketLen
	bucketTotal := uint64(bucketLen) * uint64(sizeOfHashmapBucket)
	var err error
	if _, m.bucketsOffset, err = all.alloc(bucketTotal); err != nil {
		return err
	}

	for i := uint64(0); i < uint64(m.bucketLen); i++ {
		bucket := m.byIndex(all, i)
		bucket.reset()
	}

	return nil
}

func (m *hashmap) find(all *allocator, hash uint64, key string) (prev *dataNode, node *dataNode) {
	bucket := m.byHash(all, hash)
	return bucket.find(all, key)
}

func (m *hashmap) byHash(all *allocator, hash uint64) *hashmapBucket {
	index := hash % uint64(m.bucketLen)
	return m.byIndex(all, index)
}

func (m *hashmap) byIndex(all *allocator, index uint64) *hashmapBucket {
	bucketPtr := all.base() + uintptr(index*uint64(sizeOfHashmapBucket)) + uintptr(m.bucketsOffset)
	return (*hashmapBucket)(unsafe.Pointer(bucketPtr))
}

func (m *hashmap) add(all *allocator, hash uint64, node *dataNode) {
	bucket := m.byHash(all, hash)
	bucket.add(all, node)
	m.len++
}

func (m *hashmap) delete(all *allocator, hash uint64, prev *dataNode, node *dataNode) error {
	if node == nil {
		return ErrNotFound
	}
	bucket := m.byHash(all, hash)
	bucket.delete(prev, node)
	m.len--
	return nil
}

type hashmapBucket struct {
	len               uint32
	linkedFirstOffset uint64
}

func (l *hashmapBucket) add(all *allocator, node *dataNode) {
	// 更新list链表, 头插法
	first := l.linkedFirstOffset
	// 把item的头指针指向当前的listElNode
	l.linkedFirstOffset = node.offset(all)
	// 更新next
	node.next = first
	// hashmap bucket的链表长度+1
	l.len++
}

func (l *hashmapBucket) delete(prev *dataNode, node *dataNode) {
	if prev == nil {
		// 就说明这个是头节点, 需要更新list的头节点指向
		l.linkedFirstOffset = node.next
	} else {
		prev.next = node.next
	}

	l.len--

	if l.len == 0 {
		l.linkedFirstOffset = 0
	}
}

func (l *hashmapBucket) reset() {
	*l = hashmapBucket{}
}

func (l *hashmapBucket) find(all *allocator, key string) (prev *dataNode, node *dataNode) {
	if l.len == 0 {
		return nil, nil
	}

	offset := l.linkedFirstOffset
	for i := 0; i < int(l.len); i++ {
		node = toDataNode(all, offset)
		el := nodeTo[hashmapBucketElement](node)
		if el.equal(key) {
			return
		}
		prev = node
		offset = node.next
	}
	// not found
	return nil, nil
}

// hashmapBucketElement head + lruNode + key + value
type hashmapBucketElement struct {
	keyLen uint32 // key length
	valLen uint32 // val length
}

func (el *hashmapBucketElement) reset() {
	*el = hashmapBucketElement{}
}

func (el *hashmapBucketElement) equal(key string) bool {
	if el.keyLen != uint32(len(key)) {
		return false
	}
	keyPtr := el.keyPtr()
	kh := (*reflect.StringHeader)(unsafe.Pointer(&key))
	return memequal(keyPtr, unsafe.Pointer(kh.Data), uintptr(len(key)))
}

func (el *hashmapBucketElement) updateKey(key string) {
	ss := (*reflect.StringHeader)(unsafe.Pointer(&key))
	keyPtr := el.keyPtr()
	memmove(keyPtr, unsafe.Pointer(ss.Data), uintptr(ss.Len))
	el.keyLen = uint32(len(key))
}

func (el *hashmapBucketElement) updateValue(value []byte) {
	valPtr := el.valPtr()
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&value))
	memmove(valPtr, unsafe.Pointer(bh.Data), uintptr(len(value)))
	el.valLen = uint32(len(value))
}

func (el *hashmapBucketElement) key() string {
	var s string
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	sh.Data = uintptr(el.keyPtr())
	sh.Len = int(el.keyLen)
	return s
}

func (el *hashmapBucketElement) value() []byte {
	valPtr := el.valPtr()
	var ss = make([]byte, el.valLen)
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&ss))
	memmove(unsafe.Pointer(sh.Data), valPtr, uintptr(el.valLen))
	sh.Len = int(el.valLen)
	sh.Cap = sh.Len
	return ss
}

func (el *hashmapBucketElement) lruNode() *listNode {
	ptr := uintptr(unsafe.Pointer(el)) + sizeOfHashmapBucketElement
	return (*listNode)(unsafe.Pointer(ptr))
}

func (el *hashmapBucketElement) keyPtr() unsafe.Pointer {
	//head + lruNode + key + value
	return unsafe.Pointer(uintptr(unsafe.Pointer(el)) + sizeOfHashmapBucketElement + sizeOfLRUNode)
}

func (el *hashmapBucketElement) valPtr() unsafe.Pointer {
	//head + lruNode + key + value
	ptr := uintptr(el.keyPtr()) + uintptr(el.keyLen)
	return unsafe.Pointer(ptr)
}

func hashmapElementSize(key string, value []byte) uint32 {
	return uint32(sizeOfHashmapBucketElement) + uint32(sizeOfLRUNode) + uint32(len(key)) + uint32(len(value))
}
