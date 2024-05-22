package fastcache

import (
	"testing"

	"github.com/leslie-fei/fastcache/gomemory"
	"github.com/stretchr/testify/assert"
)

var mockAllocator = newMockAllocator(uint64(128 * MB))

func newMockAllocator(size uint64) Allocator {
	mem := gomemory.NewMemory(size)
	_ = mem.Attach()
	metadata := &Metadata{
		TotalSize: size,
	}
	return &globalAllocator{mem: mem, metadata: metadata, locker: &threadLocker{}}
}

var testSize = GB

func TestCache_SetGetDelete(t *testing.T) {
	cache, err := NewCache(testSize, &Config{
		Shards:     256,
		MemoryType: GO,
		MemoryKey:  "",
	})
	assert.NoError(t, err)

	// Test Set
	err = cache.Set("key1", []byte("value1"))
	assert.NoError(t, err)

	// Test Get
	value, err := cache.Get("key1")
	assert.NoError(t, err)
	assert.Equal(t, []byte("value1"), value)

	// Test Delete
	err = cache.Delete("key1")
	assert.NoError(t, err)

	// Test Get after Delete
	value, err = cache.Get("key1")
	assert.Error(t, err)
	assert.Nil(t, value)
}

func TestCache_MultiProcess(t *testing.T) {
	cache, err := NewCache(testSize, &Config{
		Shards:     256,
		MemoryType: SHM,
		MemoryKey:  "/tmp/fastcache_test",
	})
	assert.NoError(t, err)

	// Test Set
	err = cache.Set("key1", []byte("value1"))
	assert.NoError(t, err)

	// Test Get
	value, err := cache.Get("key1")
	assert.NoError(t, err)
	assert.Equal(t, []byte("value1"), value)
}

func TestCache_Errors(t *testing.T) {
	cache, err := NewCache(testSize, &Config{
		Shards:     256,
		MemoryType: GO,
		MemoryKey:  "",
	})
	assert.NoError(t, err)

	// Test Get non-existent key
	value, err := cache.Get("non_existent")
	assert.Error(t, err)
	assert.Nil(t, value)

	// Test Delete non-existent key
	err = cache.Delete("non_existent")
	assert.Error(t, err)
}
