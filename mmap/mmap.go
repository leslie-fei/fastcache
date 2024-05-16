package mmap

import (
	"fmt"
	"os"
	"reflect"
	"unsafe"

	"github.com/edsrzf/mmap-go"
)

type Memory struct {
	filepath string
	bytes    uint64
	file     *os.File
	mmap     mmap.MMap
	basep    unsafe.Pointer
}

func NewMemory(filepath string, bytes uint64) *Memory {
	return &Memory{filepath: filepath, bytes: bytes}
}

func (m *Memory) Attach() (err error) {
	if nil == m.file {
		if m.file, err = os.OpenFile(m.filepath, os.O_RDWR|os.O_CREATE, 0666); nil != err {
			return err
		}

		if err = m.file.Truncate(int64(m.bytes)); nil != err {
			m.file.Close()
			m.file = nil
			return err
		}

		if m.mmap, err = mmap.Map(m.file, mmap.RDWR, 0); nil != err {
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
	if nil != m.mmap {
		if err := m.mmap.Unmap(); nil != err {
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
