package memlru

import (
	"errors"
	"fmt"
	"testing"

	"memlru/shm"
)

func TestMemoryManager(t *testing.T) {
	mem := shm.NewMemory("/shm/test", 128*MB, true)
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

	for i := 0; i < 1024; i++ {
		key := fmt.Sprint(i)
		if err := m.Set(key, []byte(key)); err != nil {
			t.Fatal(err)
		}

		v, err := m.Get(key)
		if err != nil {
			t.Fatal(err)
		}

		if string(v) != key {
			panic("get value not equal")
		}

		m.Del(key)
		_, err = m.Get(key)
		if !errors.Is(err, ErrNotFound) {
			t.Fatal("expect ErrNotFound")
		}
	}
}

func TestTemp(t *testing.T) {
	mem := shm.NewMemory("/shm/test", MB, true)
	if err := mem.Attach(); err != nil {
		t.Fatal(err)
	}
	memMgr := NewMemoryManager(mem)
	if err := memMgr.Init(); err != nil {
		t.Fatal(err)
	}

	m := memMgr.Hashmap()

	key := "k1"
	value := []byte("v2")

	_, _ = key, value

	if err := m.Set(key, value); err != nil {
		t.Fatal(err)
	}

	v, err := m.Get(key)
	if err != nil {
		t.Fatal(err)
	}

	if string(v) != string(value) {
		t.Fatal("get value not equal")
	}
}
