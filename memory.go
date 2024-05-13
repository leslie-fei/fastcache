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

var (
	ErrOutOfMemory = errors.New("out of memory")
)

const (
	magic                 uint64 = 9259259527
	PageSize                     = 16 * KB
	perHashmapSlotLength         = 100
	perHashmapElementSize        = 128
)

var (
	sizeOfMetadata               = unsafe.Sizeof(Metadata{})
	sizeOfHashmap                = unsafe.Sizeof(HashMap{})
	sizeOfList                   = unsafe.Sizeof(list{})
	sizeOfListElement            = unsafe.Sizeof(listElement{})
	sizeOfLinkedNode             = unsafe.Sizeof(LinkedNode{})
	sizeOfBlockFreeListContainer = unsafe.Sizeof(BlockFreeContainer{})
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
	Magic               uint64
	TotalSize           uint64
	Used                uint64
	HashMapOffset       uint64
	BlockFreeListOffset uint64
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
	mem                Memory
	metadata           *Metadata
	hashMap            *HashMap
	blockFreeContainer *BlockFreeContainer
}

func (m *MemoryManager) FreeMemory() uint64 {
	return m.metadata.TotalSize - m.metadata.Used
}

func (m *MemoryManager) Get(key string) ([]byte, error) {
	return m.hashMap.Get(m, key)
}

func (m *MemoryManager) Set(key string, value []byte) error {
	return m.hashMap.Set(m, key, value)
}

func (m *MemoryManager) Del(key string) error {
	return m.hashMap.Del(m, key)
}

func (m *MemoryManager) MaxBlockSize() uint64 {
	return m.blockFreeContainer.MaxSize()
}

func (m *MemoryManager) init() error {
	m.metadata = (*Metadata)(m.mem.Ptr())
	if m.metadata.Magic == 0 {
		m.metadata.Used = uint64(sizeOfMetadata)
		m.metadata.Magic = magic
		m.metadata.TotalSize = m.mem.Size()
		// init fixed size hashmap
		var hashPtr unsafe.Pointer
		hashPtr, m.metadata.HashMapOffset = m.alloc(uint64(sizeOfHashmap))
		hashmap := (*HashMap)(hashPtr)
		// 分配hashmap的slots array
		slots := m.calHashmapSlots()
		slotSize := slots * uint64(sizeOfList)
		_, slotOffset := m.alloc(slotSize)
		hashmap.SlotOffset = slotOffset
		hashmap.SlotLen = uint32(slots)
		// 分配block free container
		freePtr, freeOffset := m.alloc(uint64(sizeOfBlockFreeListContainer))
		freeContainer := (*BlockFreeContainer)(freePtr)
		freeContainer.Init()
		m.metadata.BlockFreeListOffset = freeOffset
	}

	if m.metadata.Magic != magic {
		return errors.New("invalid mem magic")
	}

	// 如果size变了, 需要移动数据, 并且结构中的指针都需要变动
	// 所以这里变动内存大小, 就先删除old shared memory, 重新初始化一个新的
	if m.metadata.TotalSize != m.mem.Size() {
		return errors.New("memory size changed, need to clear old shared memory")
	}

	m.hashMap = (*HashMap)(m.offset(m.metadata.HashMapOffset))
	m.blockFreeContainer = (*BlockFreeContainer)(m.offset(m.metadata.BlockFreeListOffset))

	return nil
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

func (m *MemoryManager) toLinkedNode(offset uint64) *LinkedNode {
	if offset == 0 {
		return nil
	}
	return (*LinkedNode)(m.mem.PtrOffset(offset))
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
	freeList, err := m.blockFreeContainer.Get(dataSize)
	if err != nil {
		return nil, err
	}
	// 一个节点需要的字节数等于链表头长度+定长数据长度
	fixedSize := freeList.Size
	nodeSize := uint64(sizeOfLinkedNode) + fixedSize
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
			node := (*LinkedNode)(m.offset(offset))
			node.Reset()
			// 填写数据的指针位置
			node.DataOffset = offset + uint64(sizeOfLinkedNode)
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
	freeList := m.blockFreeContainer.GetIndex(node.FreeBlockIndex)
	if freeList.Len == 0 {
		freeList.Head = node.Offset(m.basePtr())
	} else {
		node.Next = freeList.Head
		freeList.Head = node.Offset(m.basePtr())
	}
	freeList.Len++
}

func (m *MemoryManager) blockSize(node *LinkedNode) uint64 {
	freeList := m.blockFreeContainer.GetIndex(node.FreeBlockIndex)
	return freeList.Size
}
