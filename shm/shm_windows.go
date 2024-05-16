package shm

import (
	"syscall"
	"unsafe"
)

var (
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	procOpenFileMapping = kernel32.NewProc("OpenFileMappingW")
)

func openFileMapping(dwDesiredAccess uint32, bInheritHandle uint32, lpName *uint16) (syscall.Handle, error) {
	param1 := uintptr(dwDesiredAccess)
	param2 := uintptr(bInheritHandle)
	param3 := uintptr(unsafe.Pointer(lpName))
	ret, _, err := procOpenFileMapping.Call(param1, param2, param3)
	if 0 == err.(syscall.Errno) {
		err = nil
	}
	return syscall.Handle(ret), err
}

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
		name, err := syscall.UTF16PtrFromString(m.shmkey)
		if nil != err {
			return err
		}

		// open file mapping
		handle, err := openFileMapping(syscall.FILE_MAP_READ|syscall.FILE_MAP_WRITE, 0, name)
		if nil != err && m.createIfNotExists {
			// create file mapping if not exists
			sizehi := uint32(m.bytes >> 32)
			sizelo := uint32(m.bytes) & 0xffffffff
			handle, err = syscall.CreateFileMapping(syscall.InvalidHandle, nil, syscall.PAGE_READWRITE, sizehi, sizelo, name)
		}

		if nil != err {
			return err
		}

		m.shmid = uint64(handle)
	}

	// MapViewOfFile
	basep, err := syscall.MapViewOfFile(syscall.Handle(m.shmid), syscall.FILE_MAP_READ|syscall.FILE_MAP_WRITE, 0, 0, uintptr(m.bytes))
	if err != nil {
		return err
	}

	// save pointer
	m.basep = uint64(basep)
	return nil
}

func (m *Memory) Detach() (err error) {
	if m.basep != 0 {
		err = syscall.UnmapViewOfFile(uintptr(m.basep))
		m.basep = 0
	}

	return
}
