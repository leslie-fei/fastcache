package shm

import "testing"

func TestNewMemory(t *testing.T) {

	mem := NewMemory("/shm/test", 1024, true)

	if err := mem.Attach(); nil != err {
		t.Fatal(err)
	}

	p1 := (*uint32)(mem.Ptr())
	*p1 = 1234567

	p2 := (*uint32)(mem.PtrOffset(0))

	if *p1 != *p2 {
		t.Fatal("not equal:", *p1, "!=", *p2)
	}

	if err := mem.Detach(); nil != err {
		t.Fatal(err)
	}
}
