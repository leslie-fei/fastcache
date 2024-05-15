package memlru

import "unsafe"

var (
	sizeOfLRUNode = unsafe.Sizeof(listNode{})
	sizeOfLRU     = unsafe.Sizeof(list{})
)

// listNode of list
type listNode struct {
	prev     uint64 // prev listNode
	next     uint64 // next listNode
	dataNode uint64 // current DataNode offset
}

func (ln *listNode) SetPrev(memMgr *MemoryManager, node *listNode) {
	ln.prev = uint64(uintptr(unsafe.Pointer(node)) - memMgr.basePtr())
}

func (ln *listNode) SetNext(memMgr *MemoryManager, node *listNode) {
	ln.next = uint64(uintptr(unsafe.Pointer(node)) - memMgr.basePtr())
}

func (ln *listNode) Prev(memMgr *MemoryManager) *listNode {
	return (*listNode)(unsafe.Pointer(memMgr.basePtr() + uintptr(ln.prev)))
}

func (ln *listNode) Next(memMgr *MemoryManager) *listNode {
	return (*listNode)(unsafe.Pointer(memMgr.basePtr() + uintptr(ln.next)))
}

// Offset return *listNode offset base ptr
func (ln *listNode) Offset(memMgr *MemoryManager) uint64 {
	return uint64(uintptr(unsafe.Pointer(ln)) - memMgr.basePtr())
}

// list double linked list
type list struct {
	root listNode
	len  uint64
}

func (l *list) Init(memMgr *MemoryManager) {
	l.root.SetNext(memMgr, &l.root)
	l.root.SetPrev(memMgr, &l.root)
	l.len = 0
}

func (l *list) Len() uint64 {
	return l.len
}

func (l *list) Front(memMgr *MemoryManager) *listNode {
	if l.len == 0 {
		return nil
	}
	return l.root.Next(memMgr)
}

func (l *list) Back(memMgr *MemoryManager) *listNode {
	if l.len == 0 {
		return nil
	}
	return l.root.Prev(memMgr)
}

func (l *list) Remove(memMgr *MemoryManager, e *listNode) {
	l.remove(memMgr, e)
}

func (l *list) PushFront(memMgr *MemoryManager, dataNode uint64) (*listNode, error) {
	return l.insertValue(memMgr, dataNode, &l.root)
}

func (l *list) PushBack(memMgr *MemoryManager, dataNode uint64) (*listNode, error) {
	return l.insertValue(memMgr, dataNode, l.root.Prev(memMgr))
}

func (l *list) MoveToFront(memMgr *MemoryManager, e *listNode) {
	/**
	if e.list != l || l.root.next == e {
		return
	}
	l.move(e, &l.root)
	*/
	if l.root.next == e.Offset(memMgr) {
		return
	}
	l.move(memMgr, e, &l.root)
}

func (l *list) MoveToBack(memMgr *MemoryManager, e *listNode) {
	/**
	if e.list != l || l.root.prev == e {
		return
	}
	l.move(e, l.root.prev)
	*/
	if l.root.prev == e.Offset(memMgr) {
		return
	}
	l.move(memMgr, e, l.root.Prev(memMgr))
}

func (l *list) insert(memMgr *MemoryManager, e, at *listNode) *listNode {
	/**
	e.prev = at
	e.next = at.next
	e.prev.next = e
	e.next.prev = e
	e.hashmapSlot = l
	l.len++
	*/
	e.SetPrev(memMgr, at)
	e.SetNext(memMgr, at.Next(memMgr))
	e.Prev(memMgr).SetNext(memMgr, e)
	e.Next(memMgr).SetPrev(memMgr, e)
	l.len++
	return e
}

func (l *list) insertValue(memMgr *MemoryManager, dataNode uint64, at *listNode) (*listNode, error) {
	lnDataNode, err := memMgr.allocOne(uint64(sizeOfLRUNode))
	if err != nil {
		return nil, err
	}
	lruNode := NodeConvertTo[listNode](memMgr.basePtr(), lnDataNode)
	lruNode.dataNode = dataNode
	return l.insert(memMgr, lruNode, at), nil
}

func (l *list) remove(memMgr *MemoryManager, e *listNode) {
	/**
	e.prev.next = e.next
	e.next.prev = e.prev
	e.next = nil // avoid memory leaks
	e.prev = nil // avoid memory leaks
	e.list = nil
	l.len--
	*/
	e.Prev(memMgr).SetNext(memMgr, e.Next(memMgr))
	e.Next(memMgr).SetPrev(memMgr, e.Prev(memMgr))

	// free listNode
	dataNodeOffset := e.Offset(memMgr) - uint64(sizeOfDataNode)
	memMgr.free((*DataNode)(memMgr.offset(dataNodeOffset)))

	l.len--
}

func (l *list) move(memMgr *MemoryManager, e, at *listNode) {
	/**
	if e == at {
		return
	}
	e.prev.next = e.next
	e.next.prev = e.prev

	e.prev = at
	e.next = at.next
	e.prev.next = e
	e.next.prev = e
	*/
	if e == at {
		return
	}

	e.Prev(memMgr).SetNext(memMgr, e.Next(memMgr))
	e.Next(memMgr).SetPrev(memMgr, e.Prev(memMgr))
	e.SetPrev(memMgr, at)
	e.SetNext(memMgr, at.Next(memMgr))
	e.Prev(memMgr).SetNext(memMgr, e)
	e.Next(memMgr).SetPrev(memMgr, e)
}
