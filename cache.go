package fastcache

import (
	"errors"
	"fmt"
	"io"
	"unsafe"

	"github.com/leslie-fei/fastcache/gom"
	"github.com/leslie-fei/fastcache/mmap"
	"github.com/leslie-fei/fastcache/shm"
)

const (
	magic = uint64(925925925)
)

type Cache interface {
	Get(key string) ([]byte, error)
	GetWithBuffer(key string, buffer io.Writer) error
	Set(key string, value []byte) error
	Peek(key string) ([]byte, error)
	PeekWithBuffer(key string, buffer io.Writer) error
	Delete(key string) error
}

func NewCache(size int, c *Config) (Cache, error) {
	if size < 10*MB {
		return nil, ErrMemorySizeTooSmall
	}

	config := mergeConfig(size, c)
	confHash, err := getConfigHash(size, config)
	if err != nil {
		return nil, err
	}

	var mem Memory
	switch config.MemoryType {
	case GO:
		mem = gom.NewMemory(uint64(size))
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

	if err = mem.Attach(); err != nil {
		return nil, err
	}

	meta := (*metadata)(mem.Ptr())
	all := &allocator{
		mem:      mem,
		metadata: meta,
		locker:   &nopLocker{},
	}

	if meta.Magic == magic && confHash != meta.Hash {
		return nil, errors.New("config changed should remove shared memory and restart")
	}

	if meta.Magic != magic {
		meta.reset()
		meta.Magic = magic
		meta.Hash = confHash
		meta.TotalSize = mem.Size()
		meta.Used = uint64(sizeOfMetadata)

		_, lockerOffset, err := all.alloc(uint64(sizeOfProcessLocker))
		if err != nil {
			return nil, err
		}
		meta.LockerOffset = lockerOffset

		shardArrPtr, shardArrOffset, err := all.alloc(uint64(sizeOfShardArray))
		if err != nil {
			return nil, err
		}
		shrs := (*shards)(shardArrPtr)
		if err = shrs.init(all, config.Shards, config.MaxElementLen); err != nil {
			return nil, err
		}
		meta.ShardArrOffset = shardArrOffset
	}

	// 替换进程锁
	locker := (*processLocker)(unsafe.Pointer(all.base() + uintptr(meta.LockerOffset)))
	all.setLocker(locker)

	shrs := (*shards)(unsafe.Pointer(all.base() + uintptr(meta.ShardArrOffset)))
	return &cache{allocator: all, shards: shrs}, nil
}

type cache struct {
	allocator *allocator
	shards    *shards
}

func (c *cache) Peek(key string) ([]byte, error) {
	hash := xxHashString(key)
	shr := c.shard(hash)
	return shr.Peek(c.allocator, hash, key)
}

func (c *cache) PeekWithBuffer(key string, buffer io.Writer) error {
	hash := xxHashString(key)
	shr := c.shard(hash)
	return shr.PeekWithBuffer(c.allocator, hash, key, buffer)
}

func (c *cache) Delete(key string) error {
	hash := xxHashString(key)
	shr := c.shard(hash)
	return shr.Delete(c.allocator, hash, key)
}

func (c *cache) Set(key string, value []byte) error {
	hash := xxHashString(key)
	shr := c.shard(hash)
	return shr.Set(c.allocator, hash, key, value)
}

func (c *cache) Get(key string) ([]byte, error) {
	hash := xxHashString(key)
	shr := c.shard(hash)
	return shr.Get(c.allocator, hash, key)
}

func (c *cache) GetWithBuffer(key string, buffer io.Writer) error {
	hash := xxHashString(key)
	shr := c.shard(hash)
	return shr.GetWithBuffer(c.allocator, hash, key, buffer)
}

func (c *cache) shard(hash uint64) *shard {
	index := hash % uint64(c.shards.Len())
	return c.shards.shard(c.allocator, int(index))
}
