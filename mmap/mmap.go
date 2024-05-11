package mmap

import (
	"fmt"
	"os"
	"reflect"
	"syscall"
	"unsafe"
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
	if nil == m.file {
		_, err = os.Stat(m.filepath)
		if err != nil {
			if !os.IsNotExist(err) {
				return err
			}
			if m.file, err = os.Create(m.filepath); err != nil {
				return err
			}
			if err = m.file.Truncate(int64(m.bytes)); nil != err {
				m.file = nil
				return err
			}
		} else {
			if m.file, err = os.OpenFile(m.filepath, os.O_RDWR, 0666); err != nil {
				_ = m.file.Close()
				return err
			}
		}

		m.mmap, err = syscall.Mmap(int(m.file.Fd()), 0, int(m.bytes), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
		if err != nil {
			m.file = nil
			return err
		}

		sh := (*reflect.SliceHeader)(unsafe.Pointer(&m.mmap))
		m.basep = unsafe.Pointer(sh.Data)
	}
	return
}

func (m *Memory) Detach() error {
	if nil != m.mmap {
		if err := syscall.Munmap(m.mmap); nil != err {
			return err
		}
		m.mmap = nil
		m.basep = unsafe.Pointer(nil)
	}

	if nil != m.file {
		if err := m.file.Close(); nil != err {
			return err
		}
		m.file = nil
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
