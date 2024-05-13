package memlru

import (
	"errors"
	"unsafe"
)

var (
	ErrNotFound      = errors.New("key not found")
	ErrValueTooLarge = errors.New("value too large")
	ErrKeyTooLarge   = errors.New("key too large")
)

// HashMap fixed size HashMap
type HashMap struct {
	Len        uint32
	SlotLen    uint32
	SlotOffset uint64 // slots offset
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
	if len(key) > 16*KB {
		return ErrKeyTooLarge
	}

	if len(value) > int(memMgr.MaxBlockSize()) {
		return ErrValueTooLarge
	}

	item := m.item(memMgr.basePtr(), key)
	if err := item.Set(memMgr, key, value); err != nil {
		return err
	}
	// hashmap total len + 1
	m.Len++

	return nil
}

func (m *HashMap) Del(memMgr *MemoryManager, key string) error {
	item := m.item(memMgr.basePtr(), key)
	if err := item.Del(memMgr, key); err != nil {
		return err
	}
	m.Len--
	return nil
}

func (m *HashMap) item(base uintptr, key string) *list {
	hash := xxHashString(key)
	index := uint64(hash) % uint64(m.SlotLen)
	offset := index*uint64(sizeOfList) + m.SlotOffset
	return (*list)(unsafe.Pointer(base + uintptr(offset)))
}

func (m *HashMap) get(memMgr *MemoryManager, key string) (*listElement, error) {
	item := m.item(memMgr.basePtr(), key)
	find := item.Find(memMgr, key)
	if find != nil {
		return find, nil
	}
	return nil, ErrNotFound
}
