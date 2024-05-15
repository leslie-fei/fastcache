# Introduction

This is a high-performance, multi-process shared memory caching system implemented in Go. It supports Get, Set, and Delete operations, and is designed for multi-thread safety, high performance, and low latency. The cache system can be configured to use various memory allocation strategies, including Go memory, SHM, and MMap, and features zero GC.

# Features

 - support Multi-process shared memory caching
 - High performance and low latency
 - Supports various memory allocation strategies (Go memory, SHM, MMap, etc.)
 - Zero GC

# Usage

```go
    // memory of mmap
    mem := mmap.NewMemory("/tmp/TestCache", 32*MB)
    // memory of shm
    //mem := shm.NewMemory("/tmp/TestCache", 32*MB, true)
    if err := mem.Attach(); err != nil {
        panic(err)
    }
    
    defer func() {
    if err := mem.Detach(); err != nil {
        panic(err)
    }
    }()
    
    cache, err := NewCache(mem)
    if err != nil {
        panic(err)
    }
    
    k := "k1"
    v := []byte("v1")
    if err := cache.Set(k, v); err != nil {
        panic(err)
    }
    
    // if not found return ErrNotFound
    value, err := cache.Get(k)
    if err != nil {
        panic(err)
    }
    fmt.Println("get: ", string(value))
    
    err = cache.Del(k)
    if err != nil {
        panic(err)
    }
```