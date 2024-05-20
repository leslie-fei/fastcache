package fastcache

import (
	"errors"
	"reflect"
	"unsafe"
)

type shard struct {
	locker          Locker
	hashmap         *hashMap
	container       *lruAndFreeContainer
	bigContainer    *lruAndFreeContainer
	allocator       *shardAllocator
	globalAllocator *globalAllocator
}

func (s *shard) Set(hash uint64, key string, value []byte) error {
	locker := s.Locker()
	locker.Lock()
	defer locker.Unlock()
	prevNode, node, el := s.hashmap.FindNode(s.allocator.Base(), hash, key)
	isNew := node == nil
	var err error
	if isNew {
		// not found
		node, err = s.newElement(key, value)
		if err != nil {
			return err
		}
		el = NodeTo[hashMapBucketElement](node)
	} else {
		// found
		total := hashmapElementSize(key, value)
		freeList := s.container.GetIndex(node.FreeBlockIndex)
		blockSize := freeList.Size
		if total > blockSize {
			// larger than old free node size
			// new hashmapElement and delete old node
			oldNode := node
			node, err = s.newElement(key, value)
			if err != nil {
				return err
			}
			_ = s.del(hash, prevNode, oldNode)
			isNew = true
		} else {
			// update data
			el.UpdateValue(value)
		}
	}

	if isNew {
		s.hashmap.Add(s.allocator.Base(), hash, node)
	}

	// 插入到LRU list中
	if node.FreeBlockIndex > uint8(SmallFreeListIndex) {
		gLocker := s.globalAllocator.Locker()
		gLocker.Lock()
		s.bigContainer.MoveToFront(s.globalAllocator.Base(), node, &el.lruNode)
		gLocker.Unlock()
	} else {
		s.container.MoveToFront(s.globalAllocator.Base(), node, &el.lruNode)
	}

	return nil
}

func (s *shard) Get(hash uint64, key string) ([]byte, error) {
	locker := s.Locker()
	locker.RLock()
	defer locker.RUnlock()
	node, value, err := s.hashmap.Get(s.allocator.Base(), hash, key)
	if err != nil {
		return nil, err
	}

	el := NodeTo[hashMapBucketElement](node)
	if node.FreeBlockIndex > uint8(SmallFreeListIndex) {
		gLocker := s.globalAllocator.Locker()
		gLocker.Lock()
		s.bigContainer.MoveToFront(s.globalAllocator.Base(), node, &el.lruNode)
		gLocker.Unlock()
	} else {
		s.container.MoveToFront(s.globalAllocator.Base(), node, &el.lruNode)
	}

	return value, nil
}

func (s *shard) Peek(hash uint64, key string) ([]byte, error) {
	locker := s.Locker()
	locker.RLock()
	defer locker.RUnlock()
	_, v, err := s.hashmap.Get(s.allocator.Base(), hash, key)
	return v, err
}

func (s *shard) Del(hash uint64, key string) (err error) {
	locker := s.Locker()
	locker.Lock()
	defer locker.Unlock()
	prev, node, _ := s.hashmap.FindNode(s.allocator.Base(), hash, key)
	return s.del(hash, prev, node)
}

func (s *shard) Locker() Locker {
	return s.locker
}

func (s *shard) del(hash uint64, prev *DataNode, node *DataNode) error {
	err := s.hashmap.Del(s.allocator.Base(), hash, prev, node)
	if err != nil {
		return err
	}

	el := NodeTo[hashMapBucketElement](node)
	if node.FreeBlockIndex > uint8(SmallFreeListIndex) {
		// 说明是来自与bigFreeContainer
		locker := s.globalAllocator.Locker()
		locker.Lock()
		s.bigContainer.Free(s.allocator.Base(), node, &el.lruNode)
		locker.Unlock()
	} else {
		s.container.Free(s.allocator.Base(), node, &el.lruNode)
	}

	return nil
}

func (s *shard) allocOne(dataSize uint64) (*DataNode, error) {
	if dataSize == 0 {
		return nil, errors.New("data size is zero")
	}

	index := dataSizeToIndex(dataSize)
	var allocator Allocator
	var container *lruAndFreeContainer
	// 大数据块需要去全局得大数据块bigFreeContainer中获取
	if index > SmallFreeListIndex {
		allocator = s.globalAllocator
		container = s.bigContainer
	} else {
		// 小数据块直接在shard分片中直接分配, 不需要加锁, 这里的allocator Locker是一个nopLocker
		allocator = s.allocator
		container = s.container
	}
	locker := allocator.Locker()
	locker.Lock()
	defer locker.Unlock()
	node, err := container.Alloc(s.globalAllocator, dataSize)
	if err != nil {
		if errors.Is(err, ErrNoSpace) {
			// 触发淘汰
			if err = s.bigContainer.Evict(s.globalAllocator, dataSize); err != nil {
				return nil, err
			}
			// 因为已经触发过淘汰, 这里一定能拿到数据块
			node, err = s.bigContainer.Alloc(s.globalAllocator, dataSize)
			return node, err
		}
		return nil, err
	}
	return node, nil
}

func (s *shard) newElement(key string, value []byte) (*DataNode, error) {
	total := hashmapElementSize(key, value)
	elNode, err := s.allocOne(total)
	if err != nil {
		return nil, err
	}

	elNode.Len = uint32(total)
	el := NodeTo[hashMapBucketElement](elNode)
	el.keyLen = uint32(len(key))
	el.valLen = uint32(len(value))

	// set key data
	ss := (*reflect.StringHeader)(unsafe.Pointer(&key))
	keyPtr := uintptr(unsafe.Pointer(el)) + sizeOfHashmapBucketElement
	memmove(unsafe.Pointer(keyPtr), unsafe.Pointer(ss.Data), uintptr(ss.Len))

	// set value
	valPtr := keyPtr + uintptr(el.keyLen)
	// move value to DataNode data ptr
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&value))
	memmove(unsafe.Pointer(valPtr), unsafe.Pointer(bh.Data), uintptr(len(value)))

	return elNode, err
}
