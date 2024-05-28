# Introduction

This is a high-performance, multi-process shared memory caching system implemented in Go. It supports Get, Set, and Delete operations, and is designed for multi-thread safety, high performance, and low latency. The cache system can be configured to use various memory allocation strategies, including Go memory, SHM, and MMap, and features zero GC.

# Features

 - support Multi-process shared memory caching
 - High performance and low latency
 - Supports various memory allocation strategies (Go memory, SHM, MMap, etc.)
 - Zero GC
 - support LRU 

# Usage

```go
cache, err := fastcache.NewCache(fastcache.GB, &fastcache.Config{
    Shards:     sharding,
    MemoryType: fastcache.SHM, // fastcache.GO fastcache.MMAP
    MemoryKey:  "/tmp/BenchmarkFastCache_Set",
})
if err != nil {
    panic(err)
}

err = cache.Set("k1", []byte("v1"))
fmt.Println("set err: ", err)

value, err := cache.Get("k1")
fmt.Println("value: ", value, "err: ", err)
```

# Benchmark

```go
goos: windows
goarch: amd64
pkg: github.com/leslie-fei/fastcache/benchmark
cpu: Intel(R) Core(TM) i7-7700K CPU @ 4.20GHz
BenchmarkFastCache_Set
BenchmarkFastCache_Set-8         	10273144	       113.6 ns/op	       0 B/op	       0 allocs/op
BenchmarkFastCache_Get
BenchmarkFastCache_Get-8         	10430319	       113.8 ns/op	       0 B/op	       0 allocs/op
BenchmarkFastCache_SetAndGet
BenchmarkFastCache_SetAndGet-8   	10952601	       112.1 ns/op	       0 B/op	       0 allocs/op
BenchmarkBigCache_Set
BenchmarkBigCache_Set-8          	 4479901	       232.1 ns/op	    1465 B/op	       0 allocs/op
BenchmarkBigCache_Get
BenchmarkBigCache_Get-8          	12057295	       105.2 ns/op	     568 B/op	       2 allocs/op
BenchmarkBigCache_SetAndGet
BenchmarkBigCache_SetAndGet-8    	14535200	       111.0 ns/op	     638 B/op	       1 allocs/op
BenchmarkRistretto_Set
BenchmarkRistretto_Set-8         	11762238	       201.1 ns/op	     141 B/op	       3 allocs/op
BenchmarkRistretto_Get
BenchmarkRistretto_Get-8         	26380351	        64.25 ns/op	      17 B/op	       1 allocs/op
BenchmarkRistretto_SetAndGet
BenchmarkRistretto_SetAndGet-8   	21141976	        55.83 ns/op	      35 B/op	       1 allocs/op
BenchmarkTheine_Set
BenchmarkTheine_Set-8            	 2599737	       496.1 ns/op	       0 B/op	       0 allocs/op
BenchmarkTheine_Get
BenchmarkTheine_Get-8            	34772025	        30.95 ns/op	       0 B/op	       0 allocs/op
BenchmarkTheine_SetAndGet
BenchmarkTheine_SetAndGet-8      	12663650	       107.9 ns/op	       1 B/op	       0 allocs/op
PASS
```