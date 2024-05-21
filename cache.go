package fastcache

import (
	"errors"
	"fmt"
	"math"
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
	magic                  uint64 = 9259259527
	PageSize                      = 64 * KB
	perHashmapBucketLength        = 10
	perHashmapElementSize         = 32
)

var (
	sizeOfMetadata               = unsafe.Sizeof(Metadata{})
	sizeOfHashmap                = unsafe.Sizeof(hashMap{})
	sizeOfHashmapBucket          = unsafe.Sizeof(hashMapBucket{})
	sizeOfHashmapBucketElement   = unsafe.Sizeof(hashMapBucketElement{})
	sizeOfDataNode               = unsafe.Sizeof(DataNode{})
	sizeOfBlockFreeListContainer = unsafe.Sizeof(lruAndFreeContainer{})
	sizeOfShardType              = unsafe.Sizeof(shardType{})
	sizeOfBigShardType           = unsafe.Sizeof(bigShardType{})
)

type Cache interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
	Del(key string) error
	Len() uint64
	Peek(key string) ([]byte, error)
	Close() error
}

func NewCache(size int, c *Config) (Cache, error) {
	// init metadata
	if size < 10*MB {
		return nil, ErrMemorySizeTooSmall
	}

	config := mergeConfig(c)

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

	metadata := (*Metadata)(mem.Ptr())
	ga := &globalAllocator{
		mem:      mem,
		metadata: metadata,
	}
	bigBucketLen := math.Ceil(float64(config.BigDataLen) / float64(perHashmapBucketLength))
	bucketLen := math.Ceil(float64(config.MaxElementLen) / float64(config.Shards) / float64(perHashmapBucketLength))
	bigDataIndex := dataSizeToIndex(uint64(config.BigDataSize))
	// if magic not equals or memory data size changed should init memory
	reinitialize := metadata.Magic != magic || metadata.TotalSize != mem.Size()
	if reinitialize {
		metadata.Reset()
		metadata.Used = uint64(sizeOfMetadata)
		metadata.Magic = magic
		metadata.TotalSize = mem.Size()
		metadata.Shards = config.Shards

		lockerOffset, err := allocLocker(ga, config.MemoryType)
		if err != nil {
			return nil, err
		}
		metadata.LockerOffset = lockerOffset

		// big shard
		bigOffset, err := allocBigShard(ga, uint32(bigBucketLen))
		if err != nil {
			return nil, err
		}
		metadata.BigShardOffset = bigOffset
		// shards
		metadata.ShardsOffset = ga.Offset()
		for i := 0; i < int(metadata.Shards); i++ {
			if _, err = allocShard(ga, config.MemoryType, uint32(bucketLen), config.ShardPerAllocSize); err != nil {
				return nil, err
			}
		}
	}

	ga.locker = toLocker(ga, config.MemoryType, metadata.LockerOffset)

	bigType := (*bigShardType)(mem.PtrOffset(metadata.BigShardOffset))
	bigShard := toBigShard(ga, bigType)

	shards := make([]*shardProxy, 0, metadata.Shards)
	shardOffset := metadata.ShardsOffset
	for i := 0; i < int(config.Shards); i++ {
		stType := (*shardType)(unsafe.Pointer(ga.Base() + uintptr(shardOffset)))
		shr := toShard(ga, config.MemoryType, stType)
		shards = append(shards, &shardProxy{
			bigShard: bigShard,
			shard:    shr,
			bigIndex: bigDataIndex,
		})
		shardOffset += uint64(stType.Size)
	}

	return &cache{metadata: metadata, shards: shards, mem: mem}, nil
}

func mergeConfig(c *Config) *Config {
	config := DefaultConfig()
	if c != nil {
		config.MemoryKey = c.MemoryKey
		if c.MemoryType > 0 {
			config.MemoryType = c.MemoryType
		}
		if c.Shards > 0 {
			config.Shards = c.Shards
		}
		if c.MaxElementLen > 0 {
			config.MaxElementLen = c.MaxElementLen
		}
		if c.BigDataSize > 0 {
			config.BigDataSize = c.BigDataSize
		}
		if c.BigDataLen > 0 {
			config.BigDataLen = c.BigDataLen
		}
	}
	return config
}

func toBigShard(ga *globalAllocator, bigType *bigShardType) *shard {
	hashmapPtr := uintptr(bigType.HashMapOffset) + ga.Base()
	hashmap := (*hashMap)(unsafe.Pointer(hashmapPtr))
	containerPtr := uintptr(bigType.ContainerOffset) + ga.Base()
	container := (*lruAndFreeContainer)(unsafe.Pointer(containerPtr))
	return &shard{
		locker:    ga.Locker(),
		allocator: ga,
		hashmap:   hashmap,
		container: container,
	}
}

func toShard(ga *globalAllocator, memType MemoryType, shardTyp *shardType) *shard {
	locker := toLocker(ga, memType, shardTyp.LockerOffset)
	hashmapPtr := uintptr(shardTyp.HashMapOffset) + ga.Base()
	hashmap := (*hashMap)(unsafe.Pointer(hashmapPtr))
	containerPtr := uintptr(shardTyp.ContainerOffset) + ga.Base()
	container := (*lruAndFreeContainer)(unsafe.Pointer(containerPtr))

	allocator := &shardAllocator{
		global:             ga,
		shardAllocatorType: &shardTyp.Allocator,
	}

	return &shard{locker: locker, hashmap: hashmap, container: container, allocator: allocator}
}

func toLocker(ga *globalAllocator, memType MemoryType, offset uint64) Locker {
	if memType == GO {
		return (*threadLocker)(unsafe.Pointer(ga.Base() + uintptr(offset)))
	}
	return (*processLocker)(unsafe.Pointer(ga.Base() + uintptr(offset)))
}

func allocLocker(ga *globalAllocator, memType MemoryType) (offset uint64, err error) {
	if memType == GO {
		_, offset, err = ga.Alloc(uint64(unsafe.Sizeof(threadLocker{})))
		return
	}

	var ptr unsafe.Pointer
	ptr, offset, err = ga.Alloc(uint64(unsafe.Sizeof(processLocker{})))
	if err != nil {
		return
	}
	locker := (*processLocker)(ptr)
	locker.Reset()
	return
}

func allocBigShard(ga *globalAllocator, bucketLen uint32) (offset uint64, err error) {
	begin := ga.Offset()
	var ptr unsafe.Pointer
	ptr, offset, err = ga.Alloc(uint64(sizeOfBigShardType))
	if err != nil {
		return
	}
	typ := (*bigShardType)(ptr)
	var hashOffset uint64
	hashOffset, err = allocHashmap(ga, bucketLen)
	if err != nil {
		return
	}
	typ.HashMapOffset = hashOffset

	var containerOffset uint64
	containerOffset, err = allocLRUAndFreeContainer(ga)
	if err != nil {
		return
	}
	typ.ContainerOffset = containerOffset
	typ.Size = uint32(ga.Offset() - begin)
	return
}

func allocHashmap(ga *globalAllocator, bucketLen uint32) (offset uint64, err error) {
	// hashmap
	bucketSize := uint64(bucketLen) * uint64(sizeOfHashmapBucket)
	hashmapTotal := uint64(sizeOfHashmap) + bucketSize
	hashPtr, hashOffset, err := ga.Alloc(hashmapTotal)
	if err != nil {
		return 0, err
	}
	hashmap := (*hashMap)(hashPtr)
	hashmap.bucketLen = bucketLen
	return hashOffset, nil
}

func allocLRUAndFreeContainer(ga *globalAllocator) (offset uint64, err error) {
	freePtr, containerOffset, err := ga.Alloc(uint64(sizeOfBlockFreeListContainer))
	if err != nil {
		return 0, err
	}
	container := (*lruAndFreeContainer)(freePtr)
	container.Init(ga.Base())
	return containerOffset, nil
}

func allocShard(ga *globalAllocator, memType MemoryType, bucketLen uint32, perAllocSize uint64) (uint64, error) {
	// shardTypeHead + locker + hashmap + lruAndFreeContainer
	begin := ga.Offset()
	ptr, offset, err := ga.Alloc(uint64(sizeOfShardType))
	if err != nil {
		return 0, err
	}
	typ := (*shardType)(ptr)

	typ.LockerOffset, err = allocLocker(ga, memType)
	if err != nil {
		return 0, err
	}

	hashOffset, err := allocHashmap(ga, bucketLen)
	if err != nil {
		return 0, err
	}
	typ.HashMapOffset = hashOffset

	containerOffset, err := allocLRUAndFreeContainer(ga)
	if err != nil {
		return 0, err
	}
	typ.ContainerOffset = containerOffset

	typ.Allocator.growSize = perAllocSize
	typ.Size = uint32(ga.Offset() - begin)

	return offset, nil
}

type cache struct {
	mem      Memory
	metadata *Metadata
	shards   []*shardProxy
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

func (l *cache) Close() error {
	if l.mem != nil {
		return l.mem.Detach()
	}
	return nil
}

func (l *cache) shard(hash uint64) *shardProxy {
	idx := hash % uint64(len(l.shards))
	return l.shards[idx]
}
