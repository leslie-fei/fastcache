package memlru

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"syscall"
	"testing"
	"time"

	"memlru/mmap"
)

func TestMemoryManager(t *testing.T) {
	//mem := shm.NewMemory("/shm/testMemoryManager", 128*MB, true)
	mem := mmap.NewMemory("/tmp/testSharedMem", 128*MB)
	if err := mem.Attach(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := mem.Detach(); err != nil {
			t.Fatal(err)
		}
	}()

	memMgr, err := NewMemoryManager(mem)
	if err != nil {
		t.Fatal(err)
	}

	size := 1
	data := make([]byte, size)
	_, _ = rand.Read(data)

	for i := 0; i < 1024; i++ {
		key := fmt.Sprint(i)
		value := data
		if err := memMgr.init(); err != nil {
			t.Fatal("index: ", i, "err: ", err)
		}
		//key := fmt.Sprint(i)
		if err := memMgr.Set(key, value); err != nil {
			t.Fatal("index: ", i, "err: ", err)
		}

		v, err := memMgr.Get(key)
		if err != nil {
			t.Fatal(err)
		}

		if string(v) != string(value) {
			panic("get value not equal")
		}

		if err := memMgr.Del(key); err != nil {
			t.Fatal(err)
		}

		_, err = memMgr.Get(key)
		if !errors.Is(err, ErrNotFound) {
			t.Fatal("expect ErrNotFound")
		}
	}
}

func TestMMapMemory(t *testing.T) {
	filepath := "/tmp/testfile"

	var locker, err = NewFileLock("/tmp/testfile.lock")
	if err != nil {
		panic(err)
	}

	go func() {
		locker.Lock()
		mem := mmap.NewMemory(filepath, 128*MB)
		if err := mem.Attach(); err != nil {
			panic(err)
		}
		defer func() {
			if err := mem.Detach(); err != nil {
				panic(err)
			}
		}()
		memMgr, err := NewMemoryManager(mem)
		if err != nil {
			panic(err)
		}
		locker.Unlock()

		for {
			locker.Lock()
			v, err := memMgr.Get("k1")
			value := 0
			if !errors.Is(err, ErrNotFound) {
				value, _ = strconv.Atoi(string(v))
			}
			value++
			fmt.Printf("g1 v: %s err: %v\n", v, err)
			if err := memMgr.Set("k1", []byte(fmt.Sprint(value))); err != nil {
				panic(err)
			}
			locker.Unlock()
			time.Sleep(time.Second)
		}
	}()

	select {}
}

type FileLock struct {
	file *os.File
}

func NewFileLock(filename string) (*FileLock, error) {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	return &FileLock{file: file}, nil
}

func (fl *FileLock) Lock() error {
	err := syscall.Flock(int(fl.file.Fd()), syscall.LOCK_EX)
	if err != nil {
		return err
	}
	return nil
}

func (fl *FileLock) Unlock() error {
	err := syscall.Flock(int(fl.file.Fd()), syscall.LOCK_UN)
	if err != nil {
		return err
	}
	return nil
}

func (fl *FileLock) Close() error {
	return fl.file.Close()
}
