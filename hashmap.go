package memlru

import (
	"errors"
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
	if len(key) > 16*KB {
		return ErrKeyTooLarge
	}

	if len(value) > int(memMgr.MaxBlockSize()) {
		return ErrValueTooLarge
	}

	item := m.item(key)
	if err := item.Set(memMgr, key, value); err != nil {
		return err
	}
	// hashmap total len + 1
	m.Len++

	return nil
}

func (m *HashMap) Del(memMgr *MemoryManager, key string) error {
	item := m.item(key)
	if err := item.Del(memMgr, key); err != nil {
		return err
	}
	m.Len--
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
