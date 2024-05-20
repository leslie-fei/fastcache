package fastcache

import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/leslie-fei/fastcache/gomemory"
	"github.com/leslie-fei/fastcache/mmap"
	"github.com/leslie-fei/fastcache/shm"
)

var (
	ErrNoSpace            = errors.New("memory no space")
	ErrMemorySizeTooSmall = errors.New("memory size too small")
)

const (
	magic                 uint64 = 9259259527
	PageSize                     = 64 * KB
	perHashmapSlotLength         = 10
	perHashmapElementSize        = 32
)

var (
	sizeOfMetadata               = unsafe.Sizeof(Metadata{})
	sizeOfHashmap                = unsafe.Sizeof(hashMap{})
	sizeOfHashmapBucket          = unsafe.Sizeof(hashMapBucket{})
	sizeOfHashmapBucketElement   = unsafe.Sizeof(hashMapBucketElement{})
	sizeOfDataNode               = unsafe.Sizeof(DataNode{})
	sizeOfBlockFreeListContainer = unsafe.Sizeof(lruAndFreeContainer{})
)

type Cache interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
	Del(key string) error
	Len() uint64
	Peek(key string) ([]byte, error)
}

func NewCache(size int, c *Config) (Cache, error) {
	// init metadata
	if size < 10*MB {
		return nil, ErrMemorySizeTooSmall
	}

	config := DefaultConfig()
	if c != nil {
		if c.MemoryType > 0 {
			config.MemoryType = c.MemoryType
		}
		if c.Shards > 0 {
			config.Shards = c.Shards
		}
	}

	var mem Memory
	switch config.MemoryType {
	case GO:
		mem = gomemory.NewMemory(uint64(size))
	case SHM:
		if config.MemoryKey == "" {
			return nil, errors.New("shm MemoryKey is required")
		}
		mem = shm.NewMemory(config.MemoryKey, uint64(size), true)
	case MMAP:
		if config.MemoryKey == "" {
			return nil, errors.New("mmap MemoryKey is required")
		}
		mem = mmap.NewMemory(config.MemoryKey, uint64(size))
	default:
		return nil, fmt.Errorf("MemoryType: %d not support", config.MemoryType)
	}

	if err := mem.Attach(); err != nil {
		return nil, err
	}

	/**
	1. metadata 记录内存基本信息
	2. blockFreeContainer中包含1~16M的内存对象分配池
	3. 一切数据分配都通过blockFreeContainer中的FreeList来分配, 目前一共有25个FreeList, 从1~16M之间数据块分配
	4. FreeList中保存的是可用数据块, 数据块DataNode是一个单向链表
	5. 除了metadata以外的数据通过FreeList中取出DataNode进行转换, 涉及的对象都是定长对象, 简单高效
	6. 当触发空间不足需要淘汰时, 判断Set时的数据需要大小, 去对应的FreeList的LRU list中淘汰最早的用来补充这次Set
	7. cache分块, 通过shared对象, 每个shared对象包含hashmap locker etc
	8. 保证进程线程安全, 通过processLocker or threadLocker, 当时候GO内存时初始化threadLocker, 当内存是共享内存模型时就初始化processLocker
	9. processLocker就是在共享内存中通过atomic通过CAS进行, 如果CAS失败大量并发情况下退化到file lock, 保证不会死循环占用过多CPU
	*/
	// Question: 当内存不足的时候怎么进行淘汰, 一个Set对象需要包含, key DataNode, val DataNode, bucketElement{key val} DataNode
	// 都通过val的长度来, 如果申请key 1KB空间不足时, 就去淘汰一个val 1KB的节点数据
	metadata := (*Metadata)(mem.Ptr())
	// TODO global locker init
	metadata.GlobalLocker = &threadLocker{}
	metadata.GlobalLocker.Lock()
	defer metadata.GlobalLocker.Unlock()
	// new globalAllocator
	ga := &globalAllocator{
		mem:      mem,
		metadata: metadata,
	}
	// if magic not equals or memory data size changed should init memory
	reinitialize := metadata.Magic != magic || metadata.TotalSize != mem.Size()
	if reinitialize {
		metadata.Reset()
		metadata.Used = uint64(sizeOfMetadata)
		metadata.Magic = magic
		metadata.TotalSize = mem.Size()
		metadata.Shards = config.Shards
		// TODO shards memory allocator
	}
	// TODO shard and shard lockers
	// TODO 可以初始化的时候分配一个最小的bigFreeContainer, 和每个shard下的freeContainer, 确保可以保证最小的数据量进行分配淘汰
	bigFreePtr, _, err := ga.Alloc(uint64(sizeOfBlockFreeListContainer))
	if err != nil {
		return nil, err
	}
	bigFreeContainer := (*lruAndFreeContainer)(bigFreePtr)
	bigFreeContainer.Init(ga.Base())
	shards := make([]*shard, metadata.Shards)
	for i := 0; i < len(shards); i++ {
		// locker
		locker := &threadLocker{}

		// hashmap
		bucketLen := 1024
		bucketSize := uint64(bucketLen) * uint64(sizeOfHashmapBucket)
		hashPtr, _, err := ga.Alloc(uint64(sizeOfHashmap) + bucketSize)
		if err != nil {
			return nil, err
		}
		hashmap := (*hashMap)(hashPtr)
		hashmap.bucketLen = uint32(bucketLen)

		// free block list
		freePtr, _, err := ga.Alloc(uint64(sizeOfBlockFreeListContainer))
		if err != nil {
			return nil, err
		}

		freeContainer := (*lruAndFreeContainer)(freePtr)
		freeContainer.Init(ga.Base())
		allocator := &shardAllocator{
			global:   ga,
			growSize: MB,
		}

		shr := &shard{
			locker:          locker,
			hashmap:         hashmap,
			container:       freeContainer,
			allocator:       allocator,
			globalAllocator: ga,
			bigContainer:    bigFreeContainer,
		}

		shards[i] = shr
	}

	return &cache{metadata: metadata, shards: shards}, nil
}

type cache struct {
	metadata *Metadata
	shards   []*shard
	len      uint64
}

func (l *cache) Get(key string) ([]byte, error) {
	hash := xxHashString(key)
	shr := l.shard(hash)
	return shr.Get(hash, key)
}

func (l *cache) Peek(key string) ([]byte, error) {
	hash := xxHashString(key)
	shr := l.shard(hash)
	return shr.Peek(hash, key)
}

func (l *cache) Set(key string, value []byte) error {
	if len(key) > 16*KB {
		return ErrKeyTooLarge
	}

	if len(value) > 16*MB {
		return ErrValueTooLarge
	}

	hash := xxHashString(key)
	shr := l.shard(hash)

	err := shr.Set(hash, key, value)
	if err != nil {
		return err
	}
	return nil
}

func (l *cache) Del(key string) error {
	hash := xxHashString(key)
	shr := l.shard(hash)
	return shr.Del(hash, key)
}

func (l *cache) Len() uint64 {
	return l.len
}

func (l *cache) shard(hash uint64) *shard {
	idx := hash % uint64(len(l.shards))
	return l.shards[idx]
}
