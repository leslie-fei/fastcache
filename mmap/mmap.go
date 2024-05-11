package mmap

import (
	"fmt"
	"reflect"
	"syscall"
	"unsafe"
)

type Memory struct {
	filepath string
	bytes    uint64
	fd       int
	mmap     []byte
	basep    unsafe.Pointer
}

func NewMemory(filepath string, bytes uint64) *Memory {
	return &Memory{filepath: filepath, bytes: bytes}
}

func (m *Memory) Attach() (err error) {
	m.fd, err = syscall.Open(m.filepath, syscall.O_RDWR|syscall.O_CREAT|syscall.O_TRUNC, 0666)
	if err != nil {
		return err
	}

	if err = syscall.Ftruncate(m.fd, int64(m.bytes)); err != nil {
		_ = syscall.Close(m.fd)
		return err
	}

	m.mmap, err = syscall.Mmap(m.fd, 0, int(m.bytes), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		_ = syscall.Close(m.fd)
		return err
	}

	sh := (*reflect.SliceHeader)(unsafe.Pointer(&m.mmap))
	m.basep = unsafe.Pointer(sh.Data)

	return
}

func (m *Memory) Detach() error {
	if m.mmap != nil {
		if err := syscall.Munmap(m.mmap); err != nil {
			return err
		}
	}

	if m.fd > 0 {
		if err := syscall.Close(m.fd); err != nil {
			return err
		}
		m.basep = unsafe.Pointer(nil)
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
