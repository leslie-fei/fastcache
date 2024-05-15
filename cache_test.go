package fastcache

import (
	"fmt"
	"testing"
	"time"

	"github.com/leslie-fei/fastcache/mmap"
)

func TestCache(t *testing.T) {
	// memory of mmap
	mem := mmap.NewMemory("/tmp/TestCache", 32*MB)
	// memory of shm
	//mem := shm.NewMemory("/tmp/TestCache", 32*MB, true)
	if err := mem.Attach(); err != nil {
		panic(err)
	}

	defer func() {
		if err := mem.Detach(); err != nil {
			panic(err)
		}
	}()

	cache, err := NewCache(mem)
	if err != nil {
		panic(err)
	}

	k := "k1"
	v := []byte("v1")
	if err := cache.Set(k, v); err != nil {
		panic(err)
	}

	// if not found return ErrNotFound
	value, err := cache.Get(k)
	if err != nil {
		panic(err)
	}
	fmt.Println("get: ", string(value))

	err = cache.Del(k)
	if err != nil {
		panic(err)
	}
}

func TestMemoryManager_Set(t *testing.T) {
	mem := mmap.NewMemory("/tmp/TestMemoryManager_SetXXX", 16*MB)
	if err := mem.Attach(); err != nil {
		panic(err)
	}

	defer func() {
		if err := mem.Detach(); err != nil {
			panic(err)
		}
	}()

	cache, err := NewCache(mem)
	if err != nil {
		panic(err)
	}

	key := "k1"
	value := []byte("v1")
	var counter int64
	start := time.Now()
	for {
		if err := cache.Set(key, value); err != nil {
			t.Fatal(err)
		}
		counter++
		if counter >= 1000_000 {
			break
		}
	}
	elapsed := time.Since(start)
	t.Logf("QPS: %d elapsed: %s\n", int(float64(counter)/elapsed.Seconds()), elapsed)
}
