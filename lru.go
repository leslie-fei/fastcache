package fastcache

import "unsafe"

var sizeOfLRUStore = unsafe.Sizeof(lruStore{})

type lruStore struct {
	lruLists [25]list
}

func (l *lruStore) init(all *allocator) {
	for i := 0; i < len(l.lruLists); i++ {
		lruList := &l.lruLists[i]
		lruList.Init(all.base())
	}
}

func (l *lruStore) moveToFront(all *allocator, index uint8, node *listNode) {
	lruList := &l.lruLists[index]
	lruList.MoveToFront(all.base(), node)
}

func (l *lruStore) pushToFront(all *allocator, index uint8, node *listNode) {
	lruList := &l.lruLists[index]
	lruList.PushFront(all.base(), node)
}

func (l *lruStore) remove(all *allocator, index uint8, node *listNode) {
	lruList := &l.lruLists[index]
	lruList.remove(all.base(), node)
}

func (l *lruStore) get(index uint8) *list {
	return &l.lruLists[index]
}
