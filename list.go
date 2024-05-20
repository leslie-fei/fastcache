package fastcache

import "unsafe"

var (
	sizeOfLRUNode = unsafe.Sizeof(listNode{})
	sizeOfLRU     = unsafe.Sizeof(list{})
)

// listNode of list
type listNode struct {
	prev uint64 // prev listNode
	next uint64 // next listNode
	//dataNode uint64 // current DataNode offset
}

func (ln *listNode) SetPrev(base uintptr, node *listNode) {
	ln.prev = uint64(uintptr(unsafe.Pointer(node)) - base)
}

func (ln *listNode) SetNext(base uintptr, node *listNode) {
	ln.next = uint64(uintptr(unsafe.Pointer(node)) - base)
}

func (ln *listNode) Prev(base uintptr) *listNode {
	return (*listNode)(unsafe.Pointer(base + uintptr(ln.prev)))
}

func (ln *listNode) Next(base uintptr) *listNode {
	return (*listNode)(unsafe.Pointer(base + uintptr(ln.next)))
}

// Offset return *listNode offset base ptr
func (ln *listNode) Offset(base uintptr) uint64 {
	return uint64(uintptr(unsafe.Pointer(ln)) - base)
}

// list double linked list
type list struct {
	root listNode
	len  uint64
}

func (l *list) Init(base uintptr) {
	l.root.SetNext(base, &l.root)
	l.root.SetPrev(base, &l.root)
	l.len = 0
}

func (l *list) Len() uint64 {
	return l.len
}

func (l *list) Front(base uintptr) *listNode {
	if l.len == 0 {
		return nil
	}
	return l.root.Next(base)
}

func (l *list) Back(base uintptr) *listNode {
	if l.len == 0 {
		return nil
	}
	return l.root.Prev(base)
}

func (l *list) Remove(base uintptr, e *listNode) {
	l.remove(base, e)
}

func (l *list) PushFront(base uintptr, e *listNode) *listNode {
	return l.insert(base, e, &l.root)
}

func (l *list) PushBack(base uintptr, e *listNode) *listNode {
	return l.insert(base, e, l.root.Prev(base))
}

func (l *list) MoveToFront(base uintptr, e *listNode) {
	/**
	if e.list != l || l.root.next == e {
		return
	}
	l.move(e, &l.root)
	*/
	if l.root.next == e.Offset(base) {
		return
	}
	l.move(base, e, &l.root)
}

func (l *list) MoveToBack(base uintptr, e *listNode) {
	/**
	if e.list != l || l.root.prev == e {
		return
	}
	l.move(e, l.root.prev)
	*/
	if l.root.prev == e.Offset(base) {
		return
	}
	l.move(base, e, l.root.Prev(base))
}

func (l *list) insert(base uintptr, e, at *listNode) *listNode {
	/**
	e.prev = at
	e.next = at.next
	e.prev.next = e
	e.next.prev = e
	e.hashMapBucket = l
	l.len++
	*/
	e.SetPrev(base, at)
	e.SetNext(base, at.Next(base))
	e.Prev(base).SetNext(base, e)
	e.Next(base).SetPrev(base, e)
	l.len++
	return e
}

func (l *list) remove(base uintptr, e *listNode) {
	/**
	e.prev.next = e.next
	e.next.prev = e.prev
	e.next = nil // avoid memory leaks
	e.prev = nil // avoid memory leaks
	e.list = nil
	l.len--
	*/
	e.Prev(base).SetNext(base, e.Next(base))
	e.Next(base).SetPrev(base, e.Prev(base))

	l.len--
}

func (l *list) move(base uintptr, e, at *listNode) {
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

	e.Prev(base).SetNext(base, e.Next(base))
	e.Next(base).SetPrev(base, e.Prev(base))
	e.SetPrev(base, at)
	e.SetNext(base, at.Next(base))
	e.Prev(base).SetNext(base, e)
	e.Next(base).SetPrev(base, e)
}
