package fastcache

//
//import "unsafe"
//
//var sizeOfPriorityQueueValue = unsafe.Sizeof(uint64(0))
//var sizeOfPriorityQueue = unsafe.Sizeof(priorityQueue{})
//
//// priorityQueue 定长的优先队列提供给hashmapBucketElement用于TTL
//type priorityQueue struct {
//	len int64
//}
//
//func (p *priorityQueue) Len() int64 {
//	return p.len
//}
//
//func (p *priorityQueue) Less(all *allocator, i, j int) bool {
//	ie := p.indexEl(all, i)
//	je := p.indexEl(all, j)
//	return ie.expired < je.expired
//}
//
//func (p *priorityQueue) Push(all *allocator, el *hashmapBucketElement) {
//	n := p.len
//	offset := uint64(uintptr(unsafe.Pointer(el)) - all.base())
//	el.priorityIndex = n
//	// 给index赋值hashmapBucketElement offset
//	p.setIndexValue(int(n), offset)
//	p.len++
//	p.up(all, int(p.len-1))
//}
//
//func (p *priorityQueue) Pop(all *allocator) *hashmapBucketElement {
//	last := int(p.len - 1)
//	p.Swap(all, 0, last)
//	p.down(all, 0, last)
//	el := p.indexEl(all, last)
//	el.priorityIndex = -1
//	p.len--
//	return el
//}
//
//func (p *priorityQueue) Swap(all *allocator, i, j int) {
//	ie := p.indexEl(all, i)
//	je := p.indexEl(all, j)
//
//	ie.priorityIndex = int64(j)
//	je.priorityIndex = int64(i)
//
//	p.setIndexValue(i, je.offset(all))
//	p.setIndexValue(j, ie.offset(all))
//}
//
//func (p *priorityQueue) Remove(all *allocator, el *hashmapBucketElement) {
//	/**
//	n := h.Len() - 1
//	if n != i {
//		h.Swap(i, n)
//		if !down(h, i, n) {
//			up(h, i)
//		}
//	}
//	return h.Pop()
//	*/
//	n := int(p.len - 1)
//	i := int(el.priorityIndex)
//	if n != i {
//		p.Swap(all, i, n)
//		if !p.down(all, i, n) {
//			p.up(all, i)
//		}
//	}
//	p.Pop(all)
//}
//
//func (p *priorityQueue) Update(all *allocator, el *hashmapBucketElement, expired int64) {
//	p.Remove(all, el)
//	el.expired = expired
//	p.Push(all, el)
//}
//
//func (p *priorityQueue) Peek(all *allocator) *hashmapBucketElement {
//	return p.indexEl(all, 0)
//}
//
//func (p *priorityQueue) up(all *allocator, j int) {
//	for {
//		i := (j - 1) / 2 // parent
//		if i == j || !p.Less(all, j, i) {
//			break
//		}
//		p.Swap(all, i, j)
//		j = i
//	}
//}
//
//func (p *priorityQueue) down(all *allocator, i0, n int) bool {
//	i := i0
//	for {
//		j1 := 2*i + 1
//		if j1 >= n || j1 < 0 { // j1 < 0 after int overflow
//			break
//		}
//		j := j1 // left child
//		if j2 := j1 + 1; j2 < n && p.Less(all, j2, j1) {
//			j = j2 // = 2*i + 2  // right child
//		}
//		if !p.Less(all, j, i) {
//			break
//		}
//		p.Swap(all, i, j)
//		i = j
//	}
//	return i > i0
//}
//
//func (p *priorityQueue) indexEl(all *allocator, idx int) *hashmapBucketElement {
//	offset := *(*uint64)(p.indexPtr(idx))
//	return (*hashmapBucketElement)(all.offsetPtr(offset))
//}
//
//func (p *priorityQueue) indexPtr(idx int) unsafe.Pointer {
//	ptr := uintptr(unsafe.Pointer(p)) + sizeOfPriorityQueue + uintptr(idx)*sizeOfPriorityQueueValue
//	return unsafe.Pointer(ptr)
//}
//
//func (p *priorityQueue) setIndexValue(index int, value uint64) {
//	idxPtr := p.indexPtr(index)
//	*(*uint64)(idxPtr) = value
//}
