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
	magic             uint64 = 10031990
	fixedMetadataSize        = 10 * KB
	PageSize                 = 16 * KB
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

type Metadata struct {
	Magic             uint64
	TotalSize         uint64
	Used              uint64
	HashMapOffset     uint64
	Block16FreeOffset uint64
	Block1KFreeOffset uint64
	Block1MFreeOffset uint64
}

func NewMemoryManager(mem Memory) (*MemoryManager, error) {
	memMgr := &MemoryManager{
		mem: mem,
	}
	if err := memMgr.init(); err != nil {
		return nil, err
	}
	return memMgr, nil
}

type MemoryManager struct {
	metadata        *Metadata
	mem             Memory
	hashMap         *HashMap
	block16FreeList *BlockFreeList
	block1KFreeList *BlockFreeList
	block1MFreeList *BlockFreeList
}

func (m *MemoryManager) FreeMemory() uint64 {
	return m.metadata.TotalSize - m.metadata.Used
}

func (m *MemoryManager) Get(key string) ([]byte, error) {
	return m.hashMap.Get(key)
}

func (m *MemoryManager) Set(key string, value []byte) error {
	return m.hashMap.Set(key, value)
}

func (m *MemoryManager) Del(key string) error {
	return m.hashMap.Del(key)
}

func (m *MemoryManager) init() error {
	m.metadata = (*Metadata)(m.mem.Ptr())
	if m.metadata.Magic == 0 {
		m.metadata.Used = uint64(fixedMetadataSize)
		m.metadata.Magic = magic
		m.metadata.TotalSize = m.mem.Size()
		// init fixed size hashmap
		_, hashOffset := m.alloc(uint64(sizeOfHashmap))
		m.metadata.HashMapOffset = hashOffset
		_, block16Offset := m.alloc(uint64(sizeOfBlockFreeList))
		m.metadata.Block16FreeOffset = block16Offset
		_, block1KOffset := m.alloc(uint64(sizeOfBlockFreeList))
		m.metadata.Block1KFreeOffset = block1KOffset
		_, block1MOffset := m.alloc(uint64(sizeOfBlockFreeList))
		m.metadata.Block1MFreeOffset = block1MOffset
	}

	if m.metadata.Magic != magic {
		return errors.New("invalid mem magic")
	}

	m.hashMap = (*HashMap)(m.offset(m.metadata.HashMapOffset))
	m.hashMap.memMgr = m
	m.block16FreeList = (*BlockFreeList)(m.offset(m.metadata.Block16FreeOffset))
	m.block1KFreeList = (*BlockFreeList)(m.offset(m.metadata.Block1KFreeOffset))
	m.block1MFreeList = (*BlockFreeList)(m.offset(m.metadata.Block1MFreeOffset))

	return nil
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

func (m *MemoryManager) toLinkedNode(offset uint64) *LinkedNode {
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
	ptr = m.ptr()
	offset = m.metadata.Used
	m.metadata.Used += size
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
		head := freeList.First(m)
		// 头插法
		for i := 0; i < size; i++ {
			node := (*LinkedNode)(m.offset(offset))
			node.Reset()
			// 填写数据的指针位置
			node.DataOffset = offset + uint64(sizeOfLinkedNode)
			node.FreeType = int8(typ)
			if head == nil {
				head = node
			} else {
				// 头插, 把当前的head, 前面插入node节点
				next := head
				node.Next = next.Offset(m.basePtr())
				head = node
			}
			offset += uint64(nodeSize)
		}
		freeList.Len += uint32(size)
		if head != nil {
			freeList.Head = head.Offset(m.basePtr())
		}
	}

	nodes := make([]*LinkedNode, 0, num)
	for i := 0; i < num; i++ {
		// 把第一个链表节点取出
		node := (*LinkedNode)(m.offset(freeList.Head))
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
		freeList.Head = node.Offset(m.basePtr())
	} else {
		node.Next = freeList.Head
		freeList.Head = node.Offset(m.basePtr())
	}
	freeList.Len++
}
