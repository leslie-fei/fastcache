package benchmark

import (
	"context"
	"math/rand"
	_ "net/http/pprof"
	"reflect"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/Yiling-J/theine-go"
	"github.com/allegro/bigcache/v3"
	"github.com/cespare/xxhash/v2"
	"github.com/dgraph-io/ristretto"
	"github.com/leslie-fei/fastcache"
)

//go:linkname memmove runtime.memmove
//go:noescape
func memmove(dst, src unsafe.Pointer, size uintptr)

const (
	sharding   = 128
	capacity   = 100
	benchcount = 1 << 20
)

var (
	benchkeys = make([]string, 0, benchcount)
	benchVals = make([][]byte, valCount)
	valCount  = 1024
	strSource = []byte("1234567890qwertyuiopasdfghjklzxcvbnm")
)

func init() {
	for i := 0; i < benchcount; i++ {
		benchkeys = append(benchkeys, getRandStr(16))
	}

	for i := 0; i < 1024; i++ {
		r := rand.Intn(1024)
		benchVals[i] = make([]byte, r)
	}
}

func getRandStr(num int) string {
	var bb = make([]byte, num)
	for i := 0; i < num; i++ {
		bb[i] = strSource[rand.Intn(len(strSource))]
	}
	return string(bb)
}

func getIndex(i int) int {
	return i & (len(benchkeys) - 1)
}

func getValIndex(i int) int {
	return i & (valCount - 1)
}

type testShard struct {
	locker sync.RWMutex
	//hashmap map[string]uint64
	array [10000][10]byte
}

type testStruct struct {
	v1 uint32
}

func (t *testShard) set(hash uint64, key string, values []byte) {
	t.locker.Lock()
	defer t.locker.Unlock()
	idx := hash % uint64(len(t.array))
	_ = idx
	b := &t.array[idx]
	_ = b
	ts := (*testStruct)(unsafe.Pointer(b))
	ts.v1 = 1
	//sh := (*reflect.SliceHeader)(unsafe.Pointer(&values))
	//memmove(unsafe.Pointer(b), unsafe.Pointer(sh.Data), uintptr(len(values)))
}

func (t *testShard) get(hash uint64, key string) {
	t.locker.RLock()
	defer t.locker.RUnlock()
	idx := hash % uint64(len(t.array))
	_ = idx
	b := &t.array[idx]
	var rs []byte
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&rs))
	memmove(unsafe.Pointer(bh.Data), unsafe.Pointer(b), uintptr(len(t.array[idx])))
}

func BenchmarkTemporary(b *testing.B) {
	shards := make([]testShard, 128)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var i = 0
		for pb.Next() {
			index := getIndex(i)
			key := benchkeys[index]
			hash := xxhash.Sum64String(key)
			value := benchVals[getValIndex(i)]
			shard := &shards[hash%uint64(len(shards))]
			_ = value
			shard.set(hash, key, value)
			//shard.get(hash, key)
			i++
		}
	})
}

func newFastCache() fastcache.Cache {
	cache, err := fastcache.NewCache(fastcache.GB, &fastcache.Config{
		Shards:        sharding,
		MaxElementLen: 2 * benchcount,
	})
	if err != nil {
		panic(err)
	}
	return cache
}

func BenchmarkFastCache_Set(b *testing.B) {
	var mc = newFastCache()
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var i = 0
		for pb.Next() {
			index := getIndex(i)
			key := benchkeys[index]
			value := benchVals[getValIndex(i)]
			_, _, _, _ = index, value, mc, key
			//mc.Get(key)
			mc.Set(benchkeys[index], value)
			i++
		}
	})
}

func BenchmarkFastCache_Get(b *testing.B) {
	var mc = newFastCache()
	for i := 0; i < benchcount; i++ {
		value := benchVals[getValIndex(i)]
		index := getIndex(i)
		key := benchkeys[index]
		mc.Set(key, value)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var i = 0
		for pb.Next() {
			index := getIndex(i)
			_, err := mc.Get(benchkeys[index])
			if err != nil {
				panic(err)
			}
			i++
		}
	})
}

func BenchmarkFastCache_SetAndGet(b *testing.B) {
	var mc = newFastCache()
	for i := 0; i < benchcount; i++ {
		value := benchVals[getValIndex(i)]
		mc.Set(benchkeys[i%benchcount], value)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var i = 0
		for pb.Next() {
			index := getIndex(i)
			if index&7 == 0 {
				value := benchVals[getValIndex(i)]
				mc.Set(benchkeys[index], value)
			} else {
				mc.Get(benchkeys[index])
			}
			i++
		}
	})
}

func BenchmarkBigCache_Set(b *testing.B) {
	cache, _ := bigcache.New(context.Background(), bigcache.Config{
		Shards:             sharding,
		LifeWindow:         10 * time.Minute,
		MaxEntriesInWindow: 256,
		MaxEntrySize:       10000,
		Verbose:            false,
	})
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var i = 0
		for pb.Next() {
			index := getIndex(i)
			value := benchVals[getValIndex(i)]
			cache.Set(benchkeys[index], value)
			i++
		}
	})
}

func BenchmarkBigCache_Get(b *testing.B) {
	cache, _ := bigcache.New(context.Background(), bigcache.Config{
		Shards:             sharding,
		LifeWindow:         10 * time.Minute,
		MaxEntriesInWindow: 256,
		MaxEntrySize:       10000,
		Verbose:            false,
	})

	for i := 0; i < benchcount; i++ {
		value := benchVals[getValIndex(i)]
		cache.Set(benchkeys[i%benchcount], value)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var i = 0
		for pb.Next() {
			index := getIndex(i)
			cache.Get(benchkeys[index])
			i++
		}
	})
}

func BenchmarkBigCache_SetAndGet(b *testing.B) {
	cache, _ := bigcache.New(context.Background(), bigcache.Config{
		Shards:             sharding,
		LifeWindow:         10 * time.Minute,
		MaxEntriesInWindow: 256,
		MaxEntrySize:       10000,
		Verbose:            false,
	})
	for i := 0; i < benchcount; i++ {
		value := benchVals[getValIndex(i)]
		cache.Set(benchkeys[i%benchcount], value)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var i = 0
		for pb.Next() {
			index := getIndex(i)
			if index&7 == 0 {
				value := benchVals[getValIndex(i)]
				cache.Set(benchkeys[index], value)
			} else {
				cache.Get(benchkeys[index])
			}
			i++
		}
	})
}

func BenchmarkRistretto_Set(b *testing.B) {
	var mc, _ = ristretto.NewCache(&ristretto.Config{
		NumCounters: capacity * sharding * 10, // number of keys to track frequency of (10M).
		MaxCost:     1 << 30,                  // maximum cost of cache (1GB).
		BufferItems: 64,                       // number of keys per Get buffer.
	})
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var i = 0
		for pb.Next() {
			index := getIndex(i)
			value := benchVals[getValIndex(i)]
			mc.SetWithTTL(benchkeys[index], value, int64(len(value)), time.Hour)
			i++
		}
	})
}

func BenchmarkRistretto_Get(b *testing.B) {
	var mc, _ = ristretto.NewCache(&ristretto.Config{
		NumCounters: capacity * sharding * 10, // number of keys to track frequency of (10M).
		MaxCost:     1 << 30,                  // maximum cost of cache (1GB).
		BufferItems: 64,                       // number of keys per Get buffer.
	})
	for i := 0; i < benchcount; i++ {
		value := benchVals[getValIndex(i)]
		mc.SetWithTTL(benchkeys[i%benchcount], value, int64(len(value)), time.Hour)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var i = 0
		for pb.Next() {
			index := getIndex(i)
			mc.Get(benchkeys[index])
			i++
		}
	})
}

func BenchmarkRistretto_SetAndGet(b *testing.B) {
	var mc, _ = ristretto.NewCache(&ristretto.Config{
		NumCounters: capacity * sharding * 10, // number of keys to track frequency of (10M).
		MaxCost:     1 << 30,                  // maximum cost of cache (1GB).
		BufferItems: 64,                       // number of keys per Get buffer.
	})
	for i := 0; i < benchcount; i++ {
		value := benchVals[getValIndex(i)]
		mc.SetWithTTL(benchkeys[i%benchcount], value, int64(len(value)), time.Hour)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var i = 0
		for pb.Next() {
			index := getIndex(i)
			if index&7 == 0 {
				value := benchVals[getValIndex(i)]
				mc.SetWithTTL(benchkeys[index], value, int64(len(value)), time.Hour)
			} else {
				mc.Get(benchkeys[index])
			}
			i++
		}
	})
}

func BenchmarkTheine_Set(b *testing.B) {
	mc, _ := theine.NewBuilder[string, []byte](sharding * capacity).Build()
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			index := getIndex(i)
			value := benchVals[getValIndex(i)]
			i++
			mc.SetWithTTL(benchkeys[index], value, int64(len(value)), time.Hour)
		}
	})
}

func BenchmarkTheine_Get(b *testing.B) {
	mc, _ := theine.NewBuilder[string, []byte](sharding * capacity).Build()
	for i := 0; i < benchcount; i++ {
		value := benchVals[getValIndex(i)]
		mc.SetWithTTL(benchkeys[i%benchcount], value, int64(len(value)), time.Hour)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			index := getIndex(i)
			mc.Get(benchkeys[index])
			i++
		}
	})
}

func BenchmarkTheine_SetAndGet(b *testing.B) {
	mc, _ := theine.NewBuilder[string, []byte](sharding * capacity).Build()
	for i := 0; i < benchcount; i++ {
		value := benchVals[getValIndex(i)]
		mc.SetWithTTL(benchkeys[i%benchcount], value, int64(len(value)), time.Hour)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			index := getIndex(i)
			if index&7 == 0 {
				value := benchVals[getValIndex(i)]
				mc.SetWithTTL(benchkeys[index], value, int64(len(value)), time.Hour)
			} else {
				mc.Get(benchkeys[index])
			}
			i++
		}
	})
}
