package shm

import (
	"fmt"
	"unsafe"
)

// Memory 基于操作系统共享内存实现
type Memory struct {
	createIfNotExists bool   // create shm if not exists
	shmkey            string // shared memory key
	shmid             uint64 // shared memory handle
	bytes             uint64 // shared memory size
	basep             uint64 // base pointer
}

func (m *Memory) Key() string {
	return m.shmkey
}

func (m *Memory) Handle() uint64 {
	return m.shmid
}

func (m *Memory) Size() uint64 {
	return m.bytes
}

func (m *Memory) Ptr() unsafe.Pointer {
	return unsafe.Pointer(uintptr(m.basep))
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
