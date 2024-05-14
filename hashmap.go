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
	Len        uint32
	SlotLen    uint32
	SlotOffset uint64 // slots offset
}

func (m *HashMap) Get(memMgr *MemoryManager, key string) ([]byte, error) {
	item, locker := m.item(memMgr, key)
	locker.RLock()
	defer locker.RUnlock()
	find := item.Find(memMgr, key)
	if find == nil {
		return nil, ErrNotFound
	}
	node := (*DataNode)(memMgr.offset(find.ValOffset))
	return node.Data(memMgr.basePtr()), nil
}

func (m *HashMap) Set(memMgr *MemoryManager, key string, value []byte) error {
	if len(key) > 16*KB {
		return ErrKeyTooLarge
	}

	if len(value) > int(memMgr.MaxBlockSize()) {
		return ErrValueTooLarge
	}

	item, locker := m.item(memMgr, key)
	locker.Lock()
	defer locker.Unlock()
	if err := item.Set(memMgr, key, value); err != nil {
		return err
	}
	// hashmap total len + 1
	m.Len++

	return nil
}

func (m *HashMap) Del(memMgr *MemoryManager, key string) error {
	item, locker := m.item(memMgr, key)
	locker.Lock()
	defer locker.Unlock()
	if err := item.Del(memMgr, key); err != nil {
		return err
	}
	m.Len--
	return nil
}

func (m *HashMap) item(memMgr *MemoryManager, key string) (*list, *Locker) {
	hash := xxHashString(key)
	index := uint64(hash) % uint64(m.SlotLen)
	offset := index*uint64(sizeOfList) + m.SlotOffset
	lockerIdx := uint64(hash) % uint64(len(memMgr.metadata.Lockers))
	locker := &memMgr.metadata.Lockers[lockerIdx]
	return (*list)(memMgr.offset(offset)), locker
}
