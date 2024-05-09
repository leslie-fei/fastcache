package memlru

import (
	"fmt"
	"testing"

	"memlru/shm"
)

func TestMemoryManager(t *testing.T) {
	mem := shm.NewMemory("/shm/test", MB, true)
	if err := mem.Attach(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := mem.Detach(); err != nil {
			t.Fatal(err)
		}
	}()

	memMgr := NewMemoryManager(mem)
	if err := memMgr.Init(); err != nil {
		t.Fatal(err)
	}

	m := memMgr.Hashmap()

	for i := 0; i < 10; i++ {
		key := fmt.Sprint(i)
		if err := m.Set(key, []byte(key)); err != nil {
			t.Fatal(err)
		}

		v := m.Get(key)

		if string(v) != key {
			panic("get value not equal")
		}
	}
}
