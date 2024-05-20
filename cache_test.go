package fastcache

import (
	"fmt"
	"runtime"
	"testing"
)

func TestCache(t *testing.T) {
	cc, err := NewCache(128*MB, nil)
	if err != nil {
		panic(err)
	}

	var keys = make([]string, 1024)
	for i := 0; i < len(keys); i++ {
		keys[i] = fmt.Sprintf("key_%d", i)
	}

	v := []byte("v1")
	n := runtime.NumCPU()
	for i := 0; i < n; i++ {
		go func() {
			index := 0
			for {
				k := keys[index%len(keys)]
				if err := cc.Set(k, v); err != nil {
					fmt.Println("set err: ", err)
					continue
				}
				// if not found return ErrNotFound
				_, _ = cc.Get(k)
				_ = cc.Del(k)

				index++
			}
		}()
	}

	select {}
}
