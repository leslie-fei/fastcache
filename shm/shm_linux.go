package shm

import (
	"fmt"
	"hash/crc32"
	"syscall"
	"unsafe"
)

const (
	shmCreate = 01000
	shmAccess = 00600
)

func NewMemory(key string, bytes uint64, createIfNotExists bool) *Memory {
	return &Memory{
		createIfNotExists: createIfNotExists,
		shmkey:            key,
		bytes:             bytes,
	}
}

// Memory based on shm
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

func (m *Memory) Attach() error {
	if m.basep != 0 {
		return nil
	}

	if 0 == m.shmid {
		k := uintptr(crc32.ChecksumIEEE([]byte(m.shmkey)))
		s := uintptr(m.bytes)
		a := shmAccess
		if m.createIfNotExists {
			a |= shmCreate
		}

		shmid, _, errno := syscall.Syscall(syscall.SYS_SHMGET, k, s, uintptr(a))
		if errno != 0 {
			return error(errno)
		}

		m.shmid = uint64(shmid)
	}

	basep, _, errno := syscall.Syscall(syscall.SYS_SHMAT, uintptr(m.shmid), 0, 0)
	if errno != 0 {
		return error(errno)
	}

	m.basep = uint64(basep)
	return nil
}

func (m *Memory) Detach() (err error) {
	if m.basep != 0 {
		_, _, errno := syscall.Syscall(syscall.SYS_SHMDT, uintptr(m.basep), 0, 0)
		m.basep = 0
		if 0 != errno {
			err = error(errno)
		}
	}
	return
}
