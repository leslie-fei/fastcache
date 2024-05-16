package shm

import (
	"hash/crc32"
	"syscall"
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
