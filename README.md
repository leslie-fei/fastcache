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
BenchmarkTemporary
BenchmarkTemporary-8             	73428962	        15.23 ns/op
BenchmarkFastCache_Set
BenchmarkFastCache_Set-8         	14042593	        93.64 ns/op	       0 B/op	       0 allocs/op
BenchmarkFastCache_Get
BenchmarkFastCache_Get-8         	 7770978	       138.8 ns/op	     540 B/op	       0 allocs/op
BenchmarkFastCache_SetAndGet
BenchmarkFastCache_SetAndGet-8   	 9913168	       108.5 ns/op	     474 B/op	       0 allocs/op
BenchmarkBigCache_Set
BenchmarkBigCache_Set-8          	 3825472	       324.2 ns/op	    1716 B/op	       0 allocs/op
BenchmarkBigCache_Get
BenchmarkBigCache_Get-8          	 9835846	       119.8 ns/op	     556 B/op	       1 allocs/op
BenchmarkBigCache_SetAndGet
BenchmarkBigCache_SetAndGet-8    	15019317	       142.6 ns/op	     612 B/op	       1 allocs/op
BenchmarkRistretto_Set
BenchmarkRistretto_Set-8         	10617138	       202.6 ns/op	     144 B/op	       3 allocs/op
BenchmarkRistretto_Get
BenchmarkRistretto_Get-8         	26066668	        57.14 ns/op	      19 B/op	       1 allocs/op
BenchmarkRistretto_SetAndGet
BenchmarkRistretto_SetAndGet-8   	20932004	        53.77 ns/op	      34 B/op	       1 allocs/op
BenchmarkTheine_Set
BenchmarkTheine_Set-8            	 3182510	       507.1 ns/op	       0 B/op	       0 allocs/op
BenchmarkTheine_Get
BenchmarkTheine_Get-8            	31701247	        32.01 ns/op	       0 B/op	       0 allocs/op
BenchmarkTheine_SetAndGet
BenchmarkTheine_SetAndGet-8      	13708880	       104.2 ns/op	       1 B/op	       0 allocs/op
PASS

Process finished with the exit code 0
```