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

var (
	ErrOutOfMemory = errors.New("out of memory")
)

const (
	magic    = uint64(10031990)
	headSize = 10 * KB
	PageSize = 16 * KB
)

var (
	sizeOfBlockFreeList = 512 // fixed size of block free list
	sizeOfHashmap       = unsafe.Sizeof(HashMap{})
	sizeOfListElement   = unsafe.Sizeof(listElement{})
	sizeOfLinkedNode16  = unsafe.Sizeof(LinkedNode16{})
	sizeOfLinkedNode1K  = unsafe.Sizeof(LinkedNode1K{})
)

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

type LinkedNode16 struct {
	Offset uint64
	Next   uint64
	Data   [16]byte
}

type LinkedNode1K struct {
	NextOffset uint64
	Data       [KB]byte
}

func NewMemoryManager(mem Memory) *MemoryManager {
	return &MemoryManager{
		mem: mem,
	}
}

type MemoryManager struct {
	statInfo *StatInfo
	mem      Memory
	hashMap  *HashMap
}

func (m *MemoryManager) Init() error {
	m.statInfo = (*StatInfo)(m.mem.Ptr())
	if m.statInfo.Magic == 0 {
		m.statInfo.Allocated = uint64(headSize)
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

	return nil
}

func (m *MemoryManager) Offset(offset uint64) unsafe.Pointer {
	return m.mem.PtrOffset(offset)
}

func (m *MemoryManager) Ptr() unsafe.Pointer {
	return m.mem.PtrOffset(m.statInfo.Allocated)
}

func (m *MemoryManager) alloc(size uint64) (ptr unsafe.Pointer, offset uint64) {
	ptr = m.Ptr()
	offset = m.statInfo.Allocated
	m.statInfo.Allocated += size
	return
}

func (m *MemoryManager) Memory() Memory {
	return m.mem
}

func (m *MemoryManager) BasePtr() unsafe.Pointer {
	return m.mem.Ptr()
}

func (m *MemoryManager) Hashmap() *HashMap {
	return m.hashMap
}

func (m *MemoryManager) Block16FreeList() *BlockFreeList {
	return (*BlockFreeList)(m.mem.PtrOffset(m.statInfo.Block16FreeOffset))
}

func (m *MemoryManager) Block1KFreeList() *BlockFreeList {
	return (*BlockFreeList)(m.mem.PtrOffset(m.statInfo.Block1KFreeOffset))
}

func (m *MemoryManager) Block1MFreeList() *BlockFreeList {
	return (*BlockFreeList)(m.mem.PtrOffset(m.statInfo.Block1MFreeOffset))
}

func (m *MemoryManager) alloc16() (*LinkedNode16, error) {
	freeList := m.Block16FreeList()
	if freeList.Len == 0 {
		// 扩容, 申请16bytes内存链表
		if m.FreeMemory() < PageSize {
			return nil, ErrOutOfMemory
		}
		_, offset := m.alloc(PageSize)
		// 设置第一个链表节点的offset
		freeList.Head = offset
		size := PageSize / int(sizeOfLinkedNode16)
		var prev *LinkedNode16
		for i := 0; i < size; i++ {
			node := (*LinkedNode16)(m.Offset(offset))
			node.Offset = offset
			if prev == nil {
				prev = node
			} else {
				// 上一个节点的指针offset指向当前节点offset
				prev.Next = offset
				prev = node
			}
			offset += uint64(sizeOfLinkedNode16)
		}
		freeList.Len += uint32(size)
	}

	head := (*LinkedNode16)(m.Offset(freeList.Head))
	// 把第一个链表节点取出
	freeList.Head = head.Next
	freeList.Len--

	return head, nil
}

func (m *MemoryManager) free16(node *LinkedNode16) {
	// TODO free
}

func (m *MemoryManager) FreeMemory() uint64 {
	return m.statInfo.TotalSize - m.statInfo.Allocated
}
