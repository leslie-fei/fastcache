package fastcache

import (
	"errors"
	"fmt"
	"sync"
	"testing"
)

func TestCache(t *testing.T) {
	c, err := NewCache(64*MB, &Config{
		MemoryType: GO,
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
			key := []byte(fmt.Sprintf("key_%d", i))
			value := key
			err = c.Set(key, value)
			if err != nil {
				panic(err)
			}

			v, err := c.Get(key)
			if err != nil {
				panic(err)
			}

			if string(v) != string(key) {
				panic(fmt.Errorf("get key: %s value: %s not equals", key, v))
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
