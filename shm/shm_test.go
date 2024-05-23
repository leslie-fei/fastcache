package shm

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

func TestNewMemory(t *testing.T) {
	key := "/shm/test"
	size := uint64(1024)
	mem := NewMemory(key, size, true)

	if err := mem.Attach(); nil != err {
		t.Fatal(err)
	}

	p1 := (*uint32)(mem.Ptr())
	*p1 = 1234567

	p2 := (*uint32)(mem.PtrOffset(0))

	if *p1 != *p2 {
		t.Fatal("not equal:", *p1, "!=", *p2)
	}

	mem.Travel(1000, func(ptr unsafe.Pointer, size uint64) uint64 {
		_ = (*uint32)(mem.PtrOffset(0))
		return 4
	})

	assert.Equal(t, key, mem.Key())
	assert.Equal(t, size, mem.Size())
	assert.GreaterOrEqual(t, mem.Handle(), uint64(0))
	assert.Panics(t, func() {
		_ = (*uint32)(mem.PtrOffset(size + 1))
	})

	if err := mem.Detach(); nil != err {
		t.Fatal(err)
	}
}
