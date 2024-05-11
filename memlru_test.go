package memlru

import (
	"errors"
	"fmt"
	"testing"

	"memlru/mmap"
)

func TestMemoryManager(t *testing.T) {
	//mem := shm.NewMemory("/shm/testMemoryManager", 128*MB, true)
	mem := mmap.NewMemory("/tmp/testSharedMem", 128*MB)
	if err := mem.Attach(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := mem.Detach(); err != nil {
			t.Fatal(err)
		}
	}()

	memMgr, err := NewMemoryManager(mem)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 1024; i++ {
		key := fmt.Sprint(i)
		value := []byte(key)
		if err := memMgr.init(); err != nil {
			t.Fatal("index: ", i, "err: ", err)
		}
		//key := fmt.Sprint(i)
		if err := memMgr.Set(key, value); err != nil {
			t.Fatal("index: ", i, "err: ", err)
		}

		v, err := memMgr.Get(key)
		if err != nil {
			t.Fatal(err)
		}

		if string(v) != key {
			panic("get value not equal")
		}

		if err := memMgr.Del(key); err != nil {
			t.Fatal(err)
		}

		_, err = memMgr.Get(key)
		if !errors.Is(err, ErrNotFound) {
			t.Fatal("expect ErrNotFound")
		}
	}
}
