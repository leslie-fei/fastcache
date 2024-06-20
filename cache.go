package fastcache

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/leslie-fei/memcore/gom"
	"github.com/leslie-fei/memcore/mmap"
	"github.com/leslie-fei/memcore/shm"
)

const (
	magic = uint64(925925925)
)

type Cache interface {
	// Has check if the key exists in the cache.
	Has(key []byte) bool
	// HasWithCounter check if the key exists in the cache and return with counter
	HasWithCounter(key []byte) (uint8, bool)
	// GetWithCounter get key in the cache and return with counter
	GetWithCounter(key []byte) ([]byte, uint8, error)
	// GetBufferWithCounter get key with buffer and return with counter
	GetBufferWithCounter(key []byte, buffer io.Writer) (uint8, error)
	// Get value for the key, it returns ErrNotFound when key not exists
	// and LRU move to front
	Get(key []byte) ([]byte, error)
	// GetWithBuffer write value into buffer, it returns ErrNotFound when key not exists
	// and LRU move to front
	GetWithBuffer(key []byte, buffer io.Writer) error
	// Set key and value
	Set(key []byte, value []byte) error
	// Peek value for key, but it will not move LRU
	Peek(key []byte) ([]byte, error)
	// PeekWithBuffer write value into buffer, but it will not move LRU
	PeekWithBuffer(key []byte, buffer io.Writer) error
	// Delete value for key
	Delete(key []byte) error
	// Close the cache wait for the ongoing operations to complete, and return ErrCloseTimeout if timeout
	Close() error
}

type StringKeyCache interface {
	Cache
	GetStringKey(key string) ([]byte, error)
	GetStringKeyWithBuffer(key string, buffer io.Writer) error
	SetStringKey(key string, value []byte) error
	PeekStringKey(key string) ([]byte, error)
	PeekStringKeyWithBuffer(key string, buffer io.Writer) error
	DeleteStringKey(key string) error
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
		if err = allocCache(all, mem, meta, config, confHash); err != nil {
			return nil, err
		}
	}

	// 替换进程锁
	locker := (*processLocker)(unsafe.Pointer(all.base() + uintptr(meta.LockerOffset)))
	all.setLocker(locker)

	shrs := (*shards)(unsafe.Pointer(all.base() + uintptr(meta.ShardArrOffset)))
	return &cache{allocator: all, shards: shrs}, nil
}

func allocCache(all *allocator, mem Memory, meta *metadata, config *Config, confHash uint64) (err error) {
	defer func() {
		// 如果初始化就有错误, 那么重置metadata
		if err != nil {
			meta.reset()
		}
	}()
	meta.reset()
	meta.Magic = magic
	meta.Hash = confHash
	meta.TotalSize = mem.Size()
	meta.Used = uint64(sizeOfMetadata)

	_, meta.LockerOffset, err = all.alloc(uint64(sizeOfProcessLocker))
	if err != nil {
		return err
	}

	var shardArrPtr unsafe.Pointer
	shardArrPtr, meta.ShardArrOffset, err = all.alloc(uint64(sizeOfShardArray))
	if err != nil {
		return err
	}

	shrs := (*shards)(shardArrPtr)
	if err = shrs.init(all, config.Shards, config.MaxElementLen); err != nil {
		return err
	}
	return nil
}

type cache struct {
	allocator *allocator
	shards    *shards
	closed    uint32
	wg        sync.WaitGroup
	inProcess int32
}

func (c *cache) Has(key []byte) bool {
	if atomic.LoadUint32(&c.closed) == 1 {
		return false
	}
	atomic.AddInt32(&c.inProcess, 1)
	defer atomic.AddInt32(&c.inProcess, -1)
	hash := xxHashBytes(key)
	shr := c.shard(hash)
	_, ok := shr.Has(c.allocator, hash, key)
	return ok
}

func (c *cache) HasWithCounter(key []byte) (uint8, bool) {
	if atomic.LoadUint32(&c.closed) == 1 {
		return 0, false
	}
	atomic.AddInt32(&c.inProcess, 1)
	defer atomic.AddInt32(&c.inProcess, -1)
	hash := xxHashBytes(key)
	shr := c.shard(hash)
	return shr.Has(c.allocator, hash, key)
}

func (c *cache) GetWithCounter(key []byte) ([]byte, uint8, error) {
	if atomic.LoadUint32(&c.closed) == 1 {
		return nil, 0, ErrCacheClosed
	}
	atomic.AddInt32(&c.inProcess, 1)
	defer atomic.AddInt32(&c.inProcess, -1)
	hash := xxHashBytes(key)
	shr := c.shard(hash)
	return shr.Get(c.allocator, hash, key)
}

func (c *cache) GetBufferWithCounter(key []byte, buffer io.Writer) (uint8, error) {
	if atomic.LoadUint32(&c.closed) == 1 {
		return 0, ErrCacheClosed
	}
	atomic.AddInt32(&c.inProcess, 1)
	defer atomic.AddInt32(&c.inProcess, -1)
	hash := xxHashBytes(key)
	shr := c.shard(hash)
	return shr.GetWithBuffer(c.allocator, hash, key, buffer)
}

func (c *cache) Peek(key []byte) ([]byte, error) {
	if atomic.LoadUint32(&c.closed) == 1 {
		return nil, ErrCacheClosed
	}
	atomic.AddInt32(&c.inProcess, 1)
	defer atomic.AddInt32(&c.inProcess, -1)
	hash := xxHashBytes(key)
	shr := c.shard(hash)
	return shr.Peek(c.allocator, hash, key)
}

func (c *cache) PeekWithBuffer(key []byte, buffer io.Writer) error {
	if atomic.LoadUint32(&c.closed) == 1 {
		return ErrCacheClosed
	}
	atomic.AddInt32(&c.inProcess, 1)
	defer atomic.AddInt32(&c.inProcess, -1)
	hash := xxHashBytes(key)
	shr := c.shard(hash)
	return shr.PeekWithBuffer(c.allocator, hash, key, buffer)
}

func (c *cache) Delete(key []byte) error {
	if atomic.LoadUint32(&c.closed) == 1 {
		return ErrCacheClosed
	}
	atomic.AddInt32(&c.inProcess, 1)
	defer atomic.AddInt32(&c.inProcess, -1)
	hash := xxHashBytes(key)
	shr := c.shard(hash)
	return shr.Delete(c.allocator, hash, key)
}

func (c *cache) Set(key []byte, value []byte) error {
	if atomic.LoadUint32(&c.closed) == 1 {
		return ErrCacheClosed
	}
	atomic.AddInt32(&c.inProcess, 1)
	defer atomic.AddInt32(&c.inProcess, -1)
	hash := xxHashBytes(key)
	shr := c.shard(hash)
	return shr.Set(c.allocator, hash, key, value)
}

func (c *cache) Get(key []byte) ([]byte, error) {
	if atomic.LoadUint32(&c.closed) == 1 {
		return nil, ErrCacheClosed
	}
	atomic.AddInt32(&c.inProcess, 1)
	defer atomic.AddInt32(&c.inProcess, -1)
	hash := xxHashBytes(key)
	shr := c.shard(hash)
	v, _, err := shr.Get(c.allocator, hash, key)
	return v, err
}

func (c *cache) GetWithBuffer(key []byte, buffer io.Writer) error {
	if atomic.LoadUint32(&c.closed) == 1 {
		return ErrCacheClosed
	}
	atomic.AddInt32(&c.inProcess, 1)
	defer atomic.AddInt32(&c.inProcess, -1)
	hash := xxHashBytes(key)
	shr := c.shard(hash)
	_, err := shr.GetWithBuffer(c.allocator, hash, key, buffer)
	return err
}

func (c *cache) GetStringKey(key string) ([]byte, error) {
	k := s2b(key)
	return c.Get(k)
}

func (c *cache) GetStringKeyWithBuffer(key string, buffer io.Writer) error {
	k := s2b(key)
	return c.GetWithBuffer(k, buffer)
}

func (c *cache) SetStringKey(key string, value []byte) error {
	k := s2b(key)
	return c.Set(k, value)
}

func (c *cache) PeekStringKey(key string) ([]byte, error) {
	k := s2b(key)
	return c.Peek(k)
}

func (c *cache) PeekStringKeyWithBuffer(key string, buffer io.Writer) error {
	k := s2b(key)
	return c.PeekWithBuffer(k, buffer)
}

func (c *cache) DeleteStringKey(key string) error {
	k := s2b(key)
	return c.Delete(k)
}

func (c *cache) Close() error {
	if atomic.CompareAndSwapUint32(&c.closed, 0, 1) {
		time.Sleep(time.Second)
		retry := 5
		for atomic.LoadInt32(&c.inProcess) > 0 {
			if retry <= 0 {
				return ErrCloseTimeout
			}
			retry--
			time.Sleep(time.Second)
		}
	}
	return nil
}

func (c *cache) shard(hash uint64) *shard {
	index := hash % uint64(c.shards.Len())
	return c.shards.shard(c.allocator, int(index))
}
