package memlru

import (
	"crypto/rand"
	"errors"
	"fmt"
	"testing"
	"time"

	"memlru/mmap"
)

func TestMemoryManager(t *testing.T) {
	mem := mmap.NewMemory("/tmp/TestMemoryManager", 32*MB)
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

	size := 1
	data := make([]byte, size)
	_, _ = rand.Read(data)

	for i := 0; i < 1024; i++ {
		key := fmt.Sprint(i)
		value := data
		//key := fmt.Sprint(i)
		if err := cache.Set(key, value); err != nil {
			t.Fatal("index: ", i, "err: ", err)
		}

		v, err := cache.Get(key)
		if err != nil {
			t.Fatal(err)
		}

		if string(v) != string(value) {
			panic("get value not equal")
		}

		if err := cache.Del(key); err != nil {
			t.Fatal(err)
		}

		_, err = cache.Get(key)
		if !errors.Is(err, ErrNotFound) {
			t.Fatal("expect ErrNotFound")
		}
	}
}

func TestMemoryManager_Set(t *testing.T) {
	mem := mmap.NewMemory("/tmp/TestMemoryManager_SetXXX", MB)
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
