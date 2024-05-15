package memlru

import (
	"errors"
	"unsafe"
)

var (
	ErrOutOfMemory        = errors.New("out of memory")
	ErrMemorySizeTooSmall = errors.New("memory size too small")
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
	sizeOfList                   = unsafe.Sizeof(hashmapSlot{})
	sizeOfListElement            = unsafe.Sizeof(HashmapSlotElement{})
	sizeOfDataNode               = unsafe.Sizeof(DataNode{})
	sizeOfBlockFreeListContainer = unsafe.Sizeof(BlockFreeContainer{})
	sizeOfLocker                 = unsafe.Sizeof(Locker{})
)

type Cache interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
	Del(key string) error
	Len() uint64
	Peek(key string) ([]byte, error)
}

func NewCache(mem Memory) (Cache, error) {
	// init metadata
	if mem.Size() < 10*MB {
		return nil, ErrMemorySizeTooSmall
	}
	metadata := (*Metadata)(mem.Ptr())
	metadata.GlobalLocker.Lock()
	defer metadata.GlobalLocker.Unlock()
	memMgr := newMemoryManager(mem, metadata)
	// if magic not equals or memory data size changed should init memory
	needInit := metadata.Magic != magic || metadata.TotalSize != mem.Size()
	if needInit {
		metadata.Reset()
		metadata.Used = uint64(sizeOfMetadata)
		metadata.Magic = magic
		metadata.TotalSize = mem.Size()
		// init list
		var lruPtr unsafe.Pointer
		lruPtr, metadata.LRUListOffset = memMgr.alloc(uint64(sizeOfLRU))
		lruList := (*list)(lruPtr)
		lruList.Init(memMgr)
		// init fixed size hashmap
		var hashPtr unsafe.Pointer
		hashPtr, metadata.HashMapOffset = memMgr.alloc(uint64(sizeOfHashmap))
		hashmap := (*HashMap)(hashPtr)
		// 分配hashmap的slots array
		slots := memMgr.calHashmapSlots()
		slotSize := slots * uint64(sizeOfList)
		_, slotOffset := memMgr.alloc(slotSize)
		hashmap.slotOffset = slotOffset
		hashmap.slotLen = uint32(slots)
		// 分配block free container
		freePtr, freeOffset := memMgr.alloc(uint64(sizeOfBlockFreeListContainer))
		freeContainer := (*BlockFreeContainer)(freePtr)
		freeContainer.Init()
		metadata.BlockFreeContainerOffset = freeOffset
	}

	memMgr.Refresh()
	lruList := (*list)(memMgr.offset(metadata.LRUListOffset))
	hashmap := (*HashMap)(memMgr.offset(metadata.HashMapOffset))

	return &lru{metadata: metadata, list: lruList, hashMap: hashmap, memMgr: memMgr}, nil
}

type lru struct {
	metadata *Metadata
	memMgr   *MemoryManager
	list     *list
	hashMap  *HashMap
}

func (l *lru) Get(key string) ([]byte, error) {
	hash := xxHashString(key)
	locker := l.locker(hash)
	locker.RLock()
	defer locker.RUnlock()
	el, value, err := l.hashMap.Get(l.memMgr, hash, key)
	if err != nil {
		return nil, err
	}
	// move to lru list front
	lruNode := (*listNode)(l.memMgr.offset(el.lruListNodeOffset))
	l.list.MoveToFront(l.memMgr, lruNode)
	return value, nil
}

func (l *lru) Peek(key string) ([]byte, error) {
	hash := xxHashString(key)
	locker := l.locker(hash)
	locker.RLock()
	defer locker.RUnlock()
	_, value, err := l.hashMap.Get(l.memMgr, hash, key)
	return value, err
}

func (l *lru) Set(key string, value []byte) error {
	if len(key) > 16*KB {
		return ErrKeyTooLarge
	}

	if len(value) > int(l.memMgr.MaxBlockSize()) {
		return ErrValueTooLarge
	}

	hash := xxHashString(key)
	locker := l.locker(hash)
	locker.Lock()
	defer locker.Unlock()

	exists, node, err := l.hashMap.Set(l.memMgr, hash, key, value)
	if err != nil {
		return err
	}

	el := NodeConvertTo[HashmapSlotElement](l.memMgr.basePtr(), node)
	if exists {
		lruNode := (*listNode)(l.memMgr.offset(el.lruListNodeOffset))
		l.list.MoveToFront(l.memMgr, lruNode)
	} else {
		// alloc new lruNode
		newLruNode, err := l.list.PushFront(l.memMgr, node.Offset(l.memMgr.basePtr()))
		if err != nil {
			// rollback hashmap set
			_, _ = l.hashMap.Del(l.memMgr, hash, key)
			return err
		}
		// hashmap element are associated with lruNode
		el.lruListNodeOffset = newLruNode.Offset(l.memMgr)
	}

	return nil
}

func (l *lru) Del(key string) error {
	hash := xxHashString(key)
	locker := l.locker(hash)
	locker.Lock()
	defer locker.Unlock()

	el, err := l.hashMap.Del(l.memMgr, hash, key)
	if err != nil {
		return err
	}

	lruNode := (*listNode)(l.memMgr.offset(el.lruListNodeOffset))
	l.list.Remove(l.memMgr, lruNode)

	return nil
}

func (l *lru) Len() uint64 {
	return l.list.Len()
}

func (l *lru) locker(hash uint64) *Locker {
	lockerIdx := hash % uint64(len(l.metadata.Lockers))
	locker := &l.metadata.Lockers[lockerIdx]
	return locker
}
