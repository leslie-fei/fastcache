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
BenchmarkFastCache_Set-8         	27009982	        43.21 ns/op	       0 B/op	       0 allocs/op
BenchmarkFastCache_Get
BenchmarkFastCache_Get-8         	44038387	        30.38 ns/op	       0 B/op	       0 allocs/op
BenchmarkFastCache_SetAndGet
BenchmarkFastCache_SetAndGet-8   	31507309	        35.51 ns/op	       0 B/op	       0 allocs/op
BenchmarkBigCache_Set
BenchmarkBigCache_Set-8          	 5767413	       193.7 ns/op	     702 B/op	       0 allocs/op
BenchmarkBigCache_Get
BenchmarkBigCache_Get-8          	21779452	        66.17 ns/op	     255 B/op	       2 allocs/op
BenchmarkBigCache_SetAndGet
BenchmarkBigCache_SetAndGet-8    	18181873	        61.45 ns/op	     296 B/op	       1 allocs/op
BenchmarkRistretto_Set
BenchmarkRistretto_Set-8         	 2562235	       515.3 ns/op	     122 B/op	       3 allocs/op
BenchmarkRistretto_Get
BenchmarkRistretto_Get-8         	31602318	        34.99 ns/op	      17 B/op	       1 allocs/op
BenchmarkRistretto_SetAndGet
BenchmarkRistretto_SetAndGet-8   	17018234	        74.14 ns/op	      35 B/op	       1 allocs/op
BenchmarkTheine_Set
BenchmarkTheine_Set-8            	 5171096	       272.7 ns/op	       0 B/op	       0 allocs/op
BenchmarkTheine_Get
BenchmarkTheine_Get-8            	35294844	        31.94 ns/op	       0 B/op	       0 allocs/op
BenchmarkTheine_SetAndGet
BenchmarkTheine_SetAndGet-8      	15596588	        76.40 ns/op	       0 B/op	       0 allocs/op
```