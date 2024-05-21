package fastcache

import (
	"errors"
	"reflect"
	"unsafe"
)

type shardProxy struct {
	shard    *shard // 多个
	bigShard *shard // 全局
}

func (s *shardProxy) Set(hash uint64, key string, value []byte) error {
	locker := s.shard.Locker()
	locker.Lock()

	dataIndex := dataSizeToIndex(hashmapElementSize(key, value))
	isBig := dataIndex > SmallFreeListIndex
	prev, node, el := s.shard.FindNode(hash, key)
	exists := node != nil
	// 如果set的数据在小分片中存在, 并且新的value数据已经过大是大数据块, 需要把shard中的删除
	if exists && isBig {
		_ = s.shard.del(hash, prev, node)
		exists = false
	}
	if !exists {
		bigLocker := s.bigShard.Locker()
		bigLocker.Lock()
		prev, node, el = s.bigShard.FindNode(hash, key)
		if node != nil {
			// 在big中找到, 那么就应该由bigShard处理
			locker.Unlock()
			defer bigLocker.Unlock()
			return s.bigShard.Set(hash, key, value, prev, node, el)
		} else if isBig {
			// 都没有找到, 并且这个是一个大数据块, 所以这里需要放到大数据块中处理
			// 释放shard locker
			locker.Unlock()
			defer bigLocker.Unlock()
			return s.bigShard.Set(hash, key, value, prev, node, el)
		}
		// 如果bigShard中没找到, 并且不是bigdata, 那么就释放全局锁, 交给shard处理
		bigLocker.Unlock()
	}
	defer locker.Unlock()
	return s.shard.Set(hash, key, value, prev, node, el)
}

func (s *shardProxy) Get(hash uint64, key string) ([]byte, error) {
	locker := s.shard.Locker()
	locker.RLock()
	v, err := s.shard.Get(hash, key)
	locker.RUnlock()
	if err != nil && errors.Is(err, ErrNotFound) {
		bigLocker := s.bigShard.Locker()
		bigLocker.RLock()
		defer bigLocker.RUnlock()
		return s.bigShard.Get(hash, key)
	}
	return v, err
}

func (s *shardProxy) Peek(hash uint64, key string) ([]byte, error) {
	locker := s.shard.Locker()
	locker.RLock()
	v, err := s.shard.Peek(hash, key)
	locker.RUnlock()
	if err != nil && errors.Is(err, ErrNotFound) {
		bigLocker := s.bigShard.Locker()
		bigLocker.RLock()
		defer bigLocker.RUnlock()
		return s.bigShard.Peek(hash, key)
	}
	return v, err
}

func (s *shardProxy) Del(hash uint64, key string) error {
	locker := s.shard.Locker()
	locker.Lock()
	err := s.shard.Del(hash, key)
	locker.Unlock()
	if err != nil && errors.Is(err, ErrNotFound) {
		bigLocker := s.bigShard.Locker()
		bigLocker.Lock()
		defer bigLocker.Unlock()
		return s.bigShard.Del(hash, key)
	}
	return err
}

type shard struct {
	locker    Locker
	hashmap   *hashMap
	container *lruAndFreeContainer
	allocator Allocator
}

func (s *shard) FindNode(hash uint64, key string) (prev *DataNode, node *DataNode, el *hashMapBucketElement) {
	return s.hashmap.FindNode(s.allocator.Base(), hash, key)
}

func (s *shard) Set(hash uint64, key string, value []byte, prev, node *DataNode, el *hashMapBucketElement) error {
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
			_ = s.del(hash, prev, oldNode)
			isNew = true
		} else {
			// update data
			el.UpdateValue(value)
		}
	}

	if isNew {
		s.hashmap.Add(s.allocator.Base(), hash, node)
	}

	if isNew {
		s.container.PushFront(s.allocator.Base(), node, &el.lruNode)
	} else {
		s.container.MoveToFront(s.allocator.Base(), node, &el.lruNode)
	}

	return nil
}

func (s *shard) Get(hash uint64, key string) ([]byte, error) {
	node, value, err := s.hashmap.Get(s.allocator.Base(), hash, key)
	if err != nil {
		return nil, err
	}

	el := NodeTo[hashMapBucketElement](node)
	s.container.MoveToFront(s.allocator.Base(), node, &el.lruNode)

	return value, nil
}

func (s *shard) Peek(hash uint64, key string) ([]byte, error) {
	_, v, err := s.hashmap.Get(s.allocator.Base(), hash, key)
	return v, err
}

func (s *shard) Del(hash uint64, key string) (err error) {
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
	s.container.Free(s.allocator.Base(), node, &el.lruNode)

	return nil
}

func (s *shard) allocOne(dataSize uint64) (*DataNode, error) {
	if dataSize == 0 {
		return nil, errors.New("data size is zero")
	}
	// 小数据块直接在shard分片中直接分配, 不需要加锁, 这里的allocator Locker是一个nopLocker
	allocator := s.allocator
	container := s.container
	node, err := container.Alloc(allocator, dataSize)
	if err != nil {
		if errors.Is(err, ErrNoSpace) {
			// 触发淘汰
			onEvict := func(lruNode *listNode) {
				el := (*hashMapBucketElement)(unsafe.Pointer(lruNode))
				key := el.Key()
				hash := xxHashString(key)
				_ = s.Del(hash, key)
			}
			if err = container.Evict(allocator, dataSize, onEvict); err != nil {
				return nil, err
			}
			// 因为已经触发过淘汰, 这里一定能拿到数据块
			node, err = container.Alloc(allocator, dataSize)
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
