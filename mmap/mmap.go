package mmap

import (
	"fmt"
	"os"
	"reflect"
	"unsafe"

	"golang.org/x/sys/unix"
)

type Memory struct {
	filepath string
	bytes    uint64
	file     *os.File
	mmap     []byte
	basep    unsafe.Pointer
}

func NewMemory(filepath string, bytes uint64) *Memory {
	return &Memory{filepath: filepath, bytes: bytes}
}

func (m *Memory) Attach() (err error) {
	if m.file == nil {
		if m.file, err = os.OpenFile(m.filepath, os.O_RDWR|os.O_CREATE, 0666); nil != err {
			return err
		}

		if err = m.file.Truncate(int64(m.bytes)); nil != err {
			m.file.Close()
			m.file = nil
			return err
		}

		m.mmap, err = unix.Mmap(int(m.file.Fd()), 0, int(m.bytes), unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
		if err != nil {
			m.file.Close()
			m.file = nil
			return err
		}

		sh := (*reflect.SliceHeader)(unsafe.Pointer(&m.mmap))
		m.basep = unsafe.Pointer(sh.Data)
	}

	return
}

func (m *Memory) Detach() error {
	if m.mmap != nil {
		if err := unix.Munmap(m.mmap); err != nil {
			return err
		}
	}

	if m.file != nil {
		_ = m.file.Close()
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
