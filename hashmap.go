package fastcache

import (
	"errors"
	"unsafe"
)

var (
	ErrNotFound      = errors.New("key not found")
	ErrValueTooLarge = errors.New("value too large")
	ErrKeyTooLarge   = errors.New("key too large")
)

// hashMap fixed size of hashmap
type hashMap struct {
	len       uint32
	bucketLen uint32
}

func (m *hashMap) Get(base uintptr, hash uint64, key string) (*DataNode, []byte, error) {
	item := m.bucket(base, hash)
	_, findNode, findEl := item.FindNode(base, key)
	if findEl == nil {
		return nil, nil, ErrNotFound
	}
	value := findEl.Value()
	return findNode, value, nil
}

func (m *hashMap) Add(base uintptr, hash uint64, node *DataNode) {
	item := m.bucket(base, hash)
	// 更新list链表, 头插法
	item.Add(base, node)
	m.len++
}

func (m *hashMap) Del(base uintptr, hash uint64, prev *DataNode, node *DataNode) (err error) {
	item := m.bucket(base, hash)
	if err = item.Del(prev, node); err != nil {
		return
	}
	m.len--
	return
}

func (m *hashMap) FindNode(base uintptr, hash uint64, key string) (prevNode *DataNode, findNode *DataNode, findEl *hashMapBucketElement) {
	item := m.bucket(base, hash)
	return item.FindNode(base, key)
}

func (m *hashMap) bucket(base uintptr, hash uint64) *hashMapBucket {
	index := hash % uint64(m.bucketLen)
	headPtr := uintptr(unsafe.Pointer(m)) + sizeOfHashmap
	bucketPtr := uintptr(index*uint64(sizeOfHashmapBucket)) + headPtr
	return (*hashMapBucket)(unsafe.Pointer(bucketPtr))
}
