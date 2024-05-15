package memlru

import (
	"unsafe"
)

//go:linkname memmove runtime.memmove
//go:noescape
func memmove(dst, src unsafe.Pointer, size uintptr)

//go:linkname memequal runtime.memequal
//go:noescape
func memequal(a, b unsafe.Pointer, size uintptr) bool

const (
	KB = 1024
	MB = 1024 * KB
	GB = 1024 * MB
)

// Memory 内存块抽象
type Memory interface {
	// Attach attach memory
	Attach() error
	// Detach detach memory
	Detach() error
	// Ptr first ptr
	Ptr() unsafe.Pointer
	// Size memory total size
	Size() uint64
	// PtrOffset offset get ptr
	PtrOffset(offset uint64) unsafe.Pointer
	// Travel memory
	Travel(skipOffset uint64, fn func(ptr unsafe.Pointer, size uint64) uint64)
}

func newMemoryManager(mem Memory, metadata *Metadata) *MemoryManager {
	memMgr := &MemoryManager{
		mem:      mem,
		metadata: metadata,
	}
	return memMgr
}

// MemoryManager is primarily used to memory allocation
type MemoryManager struct {
	mem      Memory
	metadata *Metadata
	//hashMap            *HashMap
	blockFreeContainer *BlockFreeContainer
}

func (m *MemoryManager) Refresh() {
	m.blockFreeContainer = (*BlockFreeContainer)(m.offset(m.metadata.BlockFreeContainerOffset))
}

func (m *MemoryManager) FreeMemory() uint64 {
	return m.metadata.TotalSize - m.metadata.Used
}

func (m *MemoryManager) MaxBlockSize() uint64 {
	return m.blockFreeContainer.MaxSize()
}

func (m *MemoryManager) calHashmapSlots() uint64 {
	listSize := perHashmapSlotLength * perHashmapElementSize
	return m.mem.Size()/uint64(listSize) + 1
}

func (m *MemoryManager) offset(offset uint64) unsafe.Pointer {
	return m.mem.PtrOffset(offset)
}

func (m *MemoryManager) ptr() unsafe.Pointer {
	return m.mem.PtrOffset(m.metadata.Used)
}

func (m *MemoryManager) basePtr() uintptr {
	return uintptr(m.mem.Ptr())
}

func (m *MemoryManager) toLinkedNode(offset uint64) *DataNode {
	if offset == 0 {
		return nil
	}
	return (*DataNode)(m.mem.PtrOffset(offset))
}

// alloc memory return ptr and offset of base ptr
func (m *MemoryManager) alloc(size uint64) (ptr unsafe.Pointer, offset uint64) {
	ptr = m.ptr()
	offset = m.metadata.Used
	m.metadata.Used += size
	return
}

func (m *MemoryManager) allocOne(dataSize uint64) (*DataNode, error) {
	nodes, err := m.allocMany(1, dataSize)
	if err != nil {
		return nil, err
	}
	return nodes[0], nil
}

func (m *MemoryManager) allocMany(num int, dataSize uint64) ([]*DataNode, error) {
	freeList, err := m.blockFreeContainer.Get(dataSize)
	if err != nil {
		return nil, err
	}
	// 一个节点需要的字节数等于链表头长度+定长数据长度
	fixedSize := freeList.Size
	nodeSize := uint64(sizeOfDataNode) + fixedSize
	for freeList.Len < uint32(num) {
		allocSize := uint64(PageSize)
		if nodeSize > PageSize {
			allocSize = nodeSize
		}
		// 扩容, 申请16bytes内存链表
		if m.FreeMemory() < allocSize {
			return nil, ErrOutOfMemory
		}
		_, offset := m.alloc(allocSize)
		// 设置第一个链表节点的offset
		nodeLen := allocSize / nodeSize
		head := freeList.First(m)
		// 头插法
		for i := 0; i < int(nodeLen); i++ {
			node := (*DataNode)(m.offset(offset))
			node.Reset()
			// 填写数据的指针位置
			node.DataOffset = offset + uint64(sizeOfDataNode)
			node.FreeBlockIndex = freeList.Index
			if head == nil {
				head = node
			} else {
				// 头插, 把当前的head, 前面插入node节点
				next := head
				node.Next = next.Offset(m.basePtr())
				head = node
			}
			offset += nodeSize
		}
		freeList.Len += uint32(nodeLen)
		if head != nil {
			freeList.Head = head.Offset(m.basePtr())
		}
	}

	nodes := make([]*DataNode, 0, num)
	for i := 0; i < num; i++ {
		// 把第一个链表节点取出
		node := (*DataNode)(m.offset(freeList.Head))
		freeList.Head = node.Next
		freeList.Len--
		// 断开与这个链表的关联, 变成一个独立的node
		node.Next = 0
		nodes = append(nodes, node)
	}

	return nodes, nil
}

func (m *MemoryManager) free(node *DataNode) {
	node.Reset()
	freeList := m.blockFreeContainer.GetIndex(node.FreeBlockIndex)
	if freeList.Len == 0 {
		freeList.Head = node.Offset(m.basePtr())
	} else {
		node.Next = freeList.Head
		freeList.Head = node.Offset(m.basePtr())
	}
	freeList.Len++
}

func (m *MemoryManager) blockSize(node *DataNode) uint64 {
	freeList := m.blockFreeContainer.GetIndex(node.FreeBlockIndex)
	return freeList.Size
}
