package fastcache

import (
	"testing"
	"unsafe"

	"github.com/leslie-fei/fastcache/gomemory"
	"github.com/leslie-fei/fastcache/mmap"
	"github.com/leslie-fei/fastcache/shm"
	"github.com/stretchr/testify/assert"
)

func TestMemory(t *testing.T) {
	key := "TestMemory.test"
	size := uint64(1024)

	typs := []MemoryType{GO, SHM, MMAP}
	for _, typ := range typs {
		var mem Memory
		switch typ {
		case GO:
			mem = gomemory.NewMemory(size)
		case SHM:
			mem = shm.NewMemory(key, size, true)
		case MMAP:
			mem = mmap.NewMemory(key, size)
		}
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

		assert.Equal(t, size, mem.Size())
		assert.Panics(t, func() {
			_ = (*uint32)(mem.PtrOffset(size + 1))
		})

		if err := mem.Detach(); nil != err {
			t.Fatal(err)
		}
	}
}
