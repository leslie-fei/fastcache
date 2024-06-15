package fastcache

import (
	"errors"
	"io"
	"math"
	"unsafe"
)

var (
	sizeOfShardArray = unsafe.Sizeof(shards{})
	sizeOfShard      = unsafe.Sizeof(shard{})
)

type shards struct {
	len       uint32
	arrOffset uint64
}

func (s *shards) init(all *allocator, shardLen uint32, maxLen uint64) error {
	var err error
	size := uint64(shardLen) * uint64(sizeOfShard)
	if _, s.arrOffset, err = all.alloc(size); err != nil {
		return err
	}
	preMaxLen := uint64(math.Ceil(float64(maxLen) / float64(shardLen)))
	for i := 0; i < int(shardLen); i++ {
		shr := s.shard(all, i)
		if err = shr.init(all, preMaxLen); err != nil {
			return err
		}
	}
	s.len = shardLen
	return nil
}

func (s *shards) shard(all *allocator, index int) *shard {
	offset := uint64(index)*uint64(sizeOfShard) + s.arrOffset
	shr := (*shard)(unsafe.Pointer(all.base() + uintptr(offset)))
	return shr
}

func (s *shards) Len() uint32 {
	return s.len
}

type shard struct {
	hashmapOffset   uint64
	lruStoreOffset  uint64
	freeStoreOffset uint64
	lockerOffset    uint64
	maxLen          uint64 // 当前shard, 最大容纳数量, 超过触发LRU
}

func (s *shard) init(all *allocator, maxLen uint64) error {
	var err error
	if _, s.hashmapOffset, err = all.alloc(uint64(sizeOfHashmap)); err != nil {
		return err
	}

	hm := s.hashmap(all)
	bucketLen := nextPrime(int(math.Ceil(float64(maxLen) / 0.75)))
	if err = hm.init(all, uint32(bucketLen)); err != nil {
		return err
	}

	if _, s.lruStoreOffset, err = all.alloc(uint64(sizeOfLRUStore)); err != nil {
		return err
	}
	ls := s.lruStore(all)
	ls.init(all)

	if _, s.freeStoreOffset, err = all.alloc(uint64(sizeOfFreeStore)); err != nil {
		return err
	}
	fs := s.freeStore(all)
	if err = fs.init(all); err != nil {
		return err
	}

	if _, s.lockerOffset, err = all.alloc(uint64(sizeOfProcessLocker)); err != nil {
		return err
	}

	s.maxLen = maxLen

	return nil
}

func (s *shard) hashmap(all *allocator) *hashmap {
	return (*hashmap)(unsafe.Pointer(all.base() + uintptr(s.hashmapOffset)))
}

func (s *shard) lruStore(all *allocator) *lruStore {
	return (*lruStore)(unsafe.Pointer(all.base() + uintptr(s.lruStoreOffset)))
}

func (s *shard) freeStore(all *allocator) *freeStore {
	return (*freeStore)(unsafe.Pointer(all.base() + uintptr(s.freeStoreOffset)))
}

func (s *shard) locker(all *allocator) Locker {
	return (*processLocker)(unsafe.Pointer(all.base() + uintptr(s.lockerOffset)))
}

func (s *shard) Get(all *allocator, hash uint64, key []byte) ([]byte, error) {
	locker := s.locker(all)
	locker.Lock()
	defer locker.Unlock()

	hm := s.hashmap(all)
	_, node := hm.find(all, hash, key)
	if node == nil {
		return nil, ErrNotFound
	}

	el := nodeTo[hashmapBucketElement](node)
	value := el.value()

	ls := s.lruStore(all)
	ls.moveToFront(all, node.freeIndex, el.lruNode())

	return value, nil
}

func (s *shard) GetWithBuffer(all *allocator, hash uint64, key []byte, buffer io.Writer) error {
	locker := s.locker(all)
	locker.Lock()
	defer locker.Unlock()

	hm := s.hashmap(all)
	_, node := hm.find(all, hash, key)
	if node == nil {
		return ErrNotFound
	}

	el := nodeTo[hashmapBucketElement](node)
	if err := el.valueWithBuffer(buffer); err != nil {
		return err
	}

	ls := s.lruStore(all)
	ls.moveToFront(all, node.freeIndex, el.lruNode())
	return nil
}

func (s *shard) Peek(all *allocator, hash uint64, key []byte) ([]byte, error) {
	locker := s.locker(all)
	locker.Lock()
	defer locker.Unlock()

	hm := s.hashmap(all)
	_, node := hm.find(all, hash, key)
	if node == nil {
		return nil, ErrNotFound
	}

	el := nodeTo[hashmapBucketElement](node)
	value := el.value()

	return value, nil
}

func (s *shard) PeekWithBuffer(all *allocator, hash uint64, key []byte, buffer io.Writer) error {
	locker := s.locker(all)
	locker.Lock()
	defer locker.Unlock()

	hm := s.hashmap(all)
	_, node := hm.find(all, hash, key)
	if node == nil {
		return ErrNotFound
	}

	el := nodeTo[hashmapBucketElement](node)
	return el.valueWithBuffer(buffer)
}

func (s *shard) Set(all *allocator, hash uint64, key []byte, value []byte) error {
	locker := s.locker(all)
	locker.Lock()
	defer locker.Unlock()

	var err error
	ls := s.lruStore(all)
	hm := s.hashmap(all)
	prev, node := hm.find(all, hash, key)
	if node == nil {
		node, err = s.newElement(all, hash, key, value)
		if err != nil {
			return err
		}
		hm.add(all, hash, node)
		el := nodeTo[hashmapBucketElement](node)
		ls.pushToFront(all, node.freeIndex, el.lruNode())
	} else {
		elSize := hashmapElementSize(key, value)
		index := sizeToIndex(elSize)
		if index > node.freeIndex {
			// Delete old node and new one to replace
			old := node
			if node, err = s.newElement(all, hash, key, value); err != nil {
				return err
			}
			if err = s.del(all, hash, prev, old); err != nil {
				return err
			}
			hm.add(all, hash, node)
			el := nodeTo[hashmapBucketElement](node)
			ls.pushToFront(all, node.freeIndex, el.lruNode())
		} else {
			el := nodeTo[hashmapBucketElement](node)
			el.updateValue(value)
			ls.moveToFront(all, node.freeIndex, el.lruNode())
		}
	}

	return nil
}

func (s *shard) Delete(all *allocator, hash uint64, key []byte) error {
	locker := s.locker(all)
	locker.Lock()
	defer locker.Unlock()
	hm := s.hashmap(all)
	prev, node := hm.find(all, hash, key)
	return s.del(all, hash, prev, node)
}

func (s *shard) del(all *allocator, hash uint64, prev *dataNode, node *dataNode) error {
	hm := s.hashmap(all)
	if err := hm.delete(all, hash, prev, node); err != nil {
		return err
	}
	ls := s.lruStore(all)
	el := nodeTo[hashmapBucketElement](node)
	ls.remove(all, node.freeIndex, el.lruNode())
	fs := s.freeStore(all)
	fs.free(all, node)
	return nil
}

func (s *shard) newElement(all *allocator, hash uint64, key []byte, value []byte) (node *dataNode, err error) {
	fs := s.freeStore(all)
	elSize := hashmapElementSize(key, value)

	hm := s.hashmap(all)
	if hm.len >= s.maxLen {
		// 超过长度限制需要淘汰
		if err = s.evict(all, elSize); err != nil {
			// 有可能第一次在, 这个byte长度范围内进行分配, 就已经整体shard的长度超过了上限, 所以lru list有可能为空
			// 直接去free list里面看下有没有可以用的free node
			if !errors.Is(err, ErrLRUListIsEmpty) {
				return
			}
			err = nil
		}
	}

	node, err = fs.get(all, elSize)
	if err != nil {
		if !errors.Is(err, ErrNoSpace) {
			return
		}
		// 空间不足, 就进行淘汰
		if err = s.evict(all, elSize); err != nil {
			return
		}
		// 淘汰过后, 一定会有可用空间
		node, err = fs.get(all, elSize)
		// 如果这个时候还是没有办法得到node, 就说明有错误
		if err != nil {
			return
		}
	}

	el := nodeTo[hashmapBucketElement](node)
	el.reset()
	el.hash = hash
	el.updateKey(key)
	el.updateValue(value)
	return node, nil
}

func (s *shard) evict(all *allocator, elSize uint32) error {
	hm := s.hashmap(all)
	ls := s.lruStore(all)
	index := sizeToIndex(elSize)
	lruList := ls.get(index)
	if lruList.len == 0 {
		return ErrLRUListIsEmpty
	}
	oldest := lruList.Back(all.base())
	elPtr := uintptr(unsafe.Pointer(oldest)) - sizeOfHashmapBucketElement
	el := (*hashmapBucketElement)(unsafe.Pointer(elPtr))
	evictKey := el.key()
	hash := xxHashBytes(evictKey)
	evictPrev, evictNode := hm.find(all, hash, evictKey)
	return s.del(all, hash, evictPrev, evictNode)
}
