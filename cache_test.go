package fastcache

import (
	"errors"
	"fmt"
	"sync"
	"testing"
)

func TestCache(t *testing.T) {
	c, err := NewCache(64*MB, &Config{
		MemoryType: MMAP,
		MemoryKey:  "./cacheMMap.test",
	})
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	n := 1024
	wg.Add(n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			key := fmt.Sprintf("key_%d", i)
			err = c.Set(key, []byte(key))
			if err != nil {
				panic(err)
			}

			v, err := c.Get(key)
			if err != nil {
				panic(err)
			}

			if string(v) != key {
				panic(fmt.Errorf("Get key: %s value: %s != %s", key, v, key))
			}

			err = c.Delete(key)
			if err != nil {
				panic(err)
			}

			_, err = c.Get(key)
			if err == nil || !errors.Is(err, ErrNotFound) {
				panic("expect ErrNotFound")
			}
		}()
	}

	wg.Wait()
}
