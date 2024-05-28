package gom

import (
	"fmt"
	"reflect"
	"unsafe"
)

// Memory based on go memory
type Memory struct {
	mem   []byte
	basep unsafe.Pointer
	bytes uint64
}

func NewMemory(bytes uint64) *Memory {
	return &Memory{mem: make([]byte, bytes), bytes: bytes}
}

func (m *Memory) Attach() error {
	if nil == m.basep {
		m.mem[0] = 0
		bh := (*reflect.SliceHeader)(unsafe.Pointer(&m.mem))
		m.basep = unsafe.Pointer(bh.Data)
	}
	return nil
}

func (m *Memory) Detach() error {
	if nil != m.basep {
		m.basep = unsafe.Pointer(nil)
		m.mem = nil
	}
	return nil
}

func (m *Memory) Ptr() unsafe.Pointer {
	return m.basep
}

func (m *Memory) Size() uint64 {
	return m.bytes
}

func (m *Memory) PtrOffset(offset uint64) unsafe.Pointer {
	if offset >= m.bytes {
		panic(fmt.Errorf("offset overflow: %d > %d", offset, m.bytes))
	}
	return unsafe.Pointer(uintptr(m.basep) + uintptr(offset))
}

func (m *Memory) Travel(skipOffset uint64, fn func(ptr unsafe.Pointer, size uint64) uint64) {
	for skipOffset < m.bytes {
		if advanceBytes := fn(m.PtrOffset(skipOffset), m.bytes-skipOffset); advanceBytes > 0 {
			skipOffset += advanceBytes
			continue
		}
		break
	}
}
