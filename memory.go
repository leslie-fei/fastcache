package memlru

import (
	"errors"
	"unsafe"
)

//go:linkname memmove runtime.memmove
//go:noescape
func memmove(dst, src unsafe.Pointer, size uintptr)

const (
	KB = 1024
	MB = 1024 * KB
	GB = 1024 * MB
)

type FreeDataType int8

const (
	Free16 FreeDataType = iota
	Free1K
)

var DataSizeMap = map[FreeDataType]int{
	Free16: 16,
	Free1K: KB,
}

var (
	ErrOutOfMemory = errors.New("out of memory")
)

const (
	magic          uint64 = 10031990
	memoryHeadSize        = 10 * KB
	PageSize              = 16 * KB
)

var (
	sizeOfBlockFreeList = 512 // fixed size of block free list
	sizeOfHashmap       = unsafe.Sizeof(HashMap{})
	sizeOfListElement   = unsafe.Sizeof(listElement{})
	sizeOfLinkedNode    = unsafe.Sizeof(LinkedNode{})
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

type StatInfo struct {
	Magic             uint64
	TotalSize         uint64
	Allocated         uint64
	HashMapOffset     uint64
	Block16FreeOffset uint64
	Block1KFreeOffset uint64
	Block1MFreeOffset uint64
}

type BlockFreeList struct {
	Len  uint32
	Head uint64
}

func (bl *BlockFreeList) HeadNode(mem *MemoryManager) *LinkedNode {
	if bl.Len == 0 {
		return nil
	}
	return (*LinkedNode)(mem.Offset(bl.Head))
}

func NewMemoryManager(mem Memory) *MemoryManager {
	return &MemoryManager{
		mem: mem,
	}
}

type MemoryManager struct {
	statInfo        *StatInfo
	mem             Memory
	hashMap         *HashMap
	block16FreeList *BlockFreeList
	block1KFreeList *BlockFreeList
	block1MFreeList *BlockFreeList
}

func (m *MemoryManager) Init() error {
	m.statInfo = (*StatInfo)(m.mem.Ptr())
	if m.statInfo.Magic == 0 {
		m.statInfo.Allocated = uint64(memoryHeadSize)
		m.statInfo.Magic = magic
		m.statInfo.TotalSize = m.mem.Size()
		// init fixed size hashmap
		_, hashOffset := m.alloc(uint64(sizeOfHashmap))
		m.statInfo.HashMapOffset = hashOffset
		_, block16Offset := m.alloc(uint64(sizeOfBlockFreeList))
		m.statInfo.Block16FreeOffset = block16Offset
		_, block1KOffset := m.alloc(uint64(sizeOfBlockFreeList))
		m.statInfo.Block1KFreeOffset = block1KOffset
		_, block1MOffset := m.alloc(uint64(sizeOfBlockFreeList))
		m.statInfo.Block1MFreeOffset = block1MOffset
	}

	if m.statInfo.Magic != magic {
		return errors.New("invalid mem magic")
	}

	m.hashMap = (*HashMap)(m.Offset(m.statInfo.HashMapOffset))
	m.hashMap.memMgr = m
	m.block16FreeList = (*BlockFreeList)(m.mem.PtrOffset(m.statInfo.Block16FreeOffset))
	m.block1KFreeList = (*BlockFreeList)(m.mem.PtrOffset(m.statInfo.Block1KFreeOffset))
	m.block1MFreeList = (*BlockFreeList)(m.mem.PtrOffset(m.statInfo.Block1MFreeOffset))

	return nil
}

func (m *MemoryManager) Offset(offset uint64) unsafe.Pointer {
	return m.mem.PtrOffset(offset)
}

func (m *MemoryManager) Ptr() unsafe.Pointer {
	return m.mem.PtrOffset(m.statInfo.Allocated)
}

func (m *MemoryManager) Memory() Memory {
	return m.mem
}

func (m *MemoryManager) BasePtr() uintptr {
	return uintptr(m.mem.Ptr())
}

func (m *MemoryManager) Hashmap() *HashMap {
	return m.hashMap
}

func (m *MemoryManager) FreeMemory() uint64 {
	return m.statInfo.TotalSize - m.statInfo.Allocated
}

func (m *MemoryManager) ToLinkedNode(offset uint64) *LinkedNode {
	if offset == 0 {
		return nil
	}
	return (*LinkedNode)(m.mem.PtrOffset(offset))
}

func (m *MemoryManager) selectFreeBySize(dataSize uint64) (*BlockFreeList, FreeDataType) {
	typ := Free1K
	if dataSize < 16 {
		typ = Free16
	}
	return m.selectFreeByType(typ), typ
}

func (m *MemoryManager) selectFreeByType(typ FreeDataType) *BlockFreeList {
	switch typ {
	case Free16:
		return m.block16FreeList
	case Free1K:
		return m.block1KFreeList
	default:
		return m.block1KFreeList
	}
}

// alloc memory return ptr and offset of base ptr
func (m *MemoryManager) alloc(size uint64) (ptr unsafe.Pointer, offset uint64) {
	ptr = m.Ptr()
	offset = m.statInfo.Allocated
	m.statInfo.Allocated += size
	return
}

func (m *MemoryManager) allocOne(dataSize uint64) (*LinkedNode, error) {
	nodes, err := m.allocMany(1, dataSize)
	if err != nil {
		return nil, err
	}
	return nodes[0], nil
}

func (m *MemoryManager) allocMany(num int, dataSize uint64) ([]*LinkedNode, error) {
	freeList, typ := m.selectFreeBySize(dataSize)
	// 一个节点需要的字节数等于链表头长度+定长数据长度
	fixedSize := DataSizeMap[typ]
	nodeSize := int(sizeOfLinkedNode) + fixedSize
	for freeList.Len < uint32(num) {
		// 扩容, 申请16bytes内存链表
		if m.FreeMemory() < PageSize {
			return nil, ErrOutOfMemory
		}
		_, offset := m.alloc(PageSize)
		// 设置第一个链表节点的offset
		size := PageSize / int(nodeSize)
		head := freeList.HeadNode(m)
		// 头插法
		for i := 0; i < size; i++ {
			node := (*LinkedNode)(m.Offset(offset))
			node.Reset()
			// 填写数据的指针位置
			node.DataOffset = offset + uint64(sizeOfLinkedNode)
			node.FreeType = int8(typ)
			if head == nil {
				head = node
			} else {
				// 头插, 把当前的head, 前面插入node节点
				next := head
				node.Next = next.Offset(m.BasePtr())
				head = node
			}
			offset += uint64(nodeSize)
		}
		freeList.Len += uint32(size)
		if head != nil {
			freeList.Head = head.Offset(m.BasePtr())
		}
	}

	nodes := make([]*LinkedNode, 0, num)
	for i := 0; i < num; i++ {
		// 把第一个链表节点取出
		node := (*LinkedNode)(m.Offset(freeList.Head))
		freeList.Head = node.Next
		freeList.Len--
		// 断开与这个链表的关联, 变成一个独立的node
		node.Next = 0
		nodes = append(nodes, node)
	}

	return nodes, nil
}

func (m *MemoryManager) free(node *LinkedNode) {
	node.Reset()
	freeList := m.selectFreeByType(FreeDataType(node.FreeType))
	if freeList.Len == 0 {
		freeList.Head = node.Offset(m.BasePtr())
	} else {
		node.Next = freeList.Head
		freeList.Head = node.Offset(m.BasePtr())
	}
	freeList.Len++
}
