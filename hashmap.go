package fastcache

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
	len        uint32
	slotLen    uint32
	slotOffset uint64 // slots offset
}

func (m *HashMap) Get(memMgr *MemoryManager, hash uint64, key string) (*HashmapSlotElement, []byte, error) {
	item := m.slot(memMgr, hash)
	findEl := item.Find(memMgr, key)
	if findEl == nil {
		return nil, nil, ErrNotFound
	}
	node := (*DataNode)(memMgr.offset(findEl.valOffset))
	value := node.Data(memMgr.basePtr())
	return findEl, value, nil
}

func (m *HashMap) Set(memMgr *MemoryManager, hash uint64, key string, value []byte) (exists bool, node *DataNode, err error) {
	item := m.slot(memMgr, hash)
	exists, node, err = item.Set(memMgr, key, value)
	if err != nil {
		return
	}
	// if is new, hashmap total len + 1
	if !exists {
		m.len++
	}

	return
}

func (m *HashMap) Del(memMgr *MemoryManager, hash uint64, key string) (el *HashmapSlotElement, err error) {
	item := m.slot(memMgr, hash)
	if el, err = item.Del(memMgr, key); err != nil {
		return
	}
	m.len--
	return
}

func (m *HashMap) slot(memMgr *MemoryManager, hash uint64) *hashmapSlot {
	index := hash % uint64(m.slotLen)
	offset := index*uint64(sizeOfList) + m.slotOffset
	return (*hashmapSlot)(memMgr.offset(offset))
}
