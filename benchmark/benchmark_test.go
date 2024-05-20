package benchmark

import (
	_ "net/http/pprof"
	"testing"
	"time"

	"github.com/Yiling-J/theine-go"
	"github.com/dgraph-io/ristretto"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/leslie-fei/fastcache"
	"github.com/leslie-fei/fastcache/benchmark/utils"
	"github.com/lxzan/memorycache"
	"github.com/stretchr/testify/assert"
)

const (
	sharding = 128
	capacity = 100
	//benchcount = 1 << 20
	benchcount = 1 << 10
)

var (
	benchkeys = make([]string, 0, benchcount)

	options = []memorycache.Option{
		memorycache.WithBucketNum(sharding),
		memorycache.WithBucketSize(capacity/10, capacity),
		memorycache.WithSwissTable(false),
	}
)

func init() {
	for i := 0; i < benchcount; i++ {
		benchkeys = append(benchkeys, string(utils.AlphabetNumeric.Generate(16)))
	}
	//go func() {
	//	if err := http.ListenAndServe(":6060", nil); err != nil {
	//		panic(err)
	//	}
	//}()
}

func getIndex(i int) int {
	return i & (len(benchkeys) - 1)
}

func BenchmarkFastCache_Set(b *testing.B) {
	cache, err := fastcache.NewCache(fastcache.GB, nil)
	if err != nil {
		b.Fatal(err)
	}
	var mc = cache
	var value = []byte{1}
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var i = 0
		for pb.Next() {
			index := getIndex(i)
			i++
			mc.Set(benchkeys[index], value)
		}
	})
}

func BenchmarkFastCache_Get(b *testing.B) {
	cache, err := fastcache.NewCache(fastcache.GB, nil)
	if err != nil {
		b.Fatal(err)
	}
	mc := cache
	var value = []byte("1")
	for i := 0; i < benchcount; i++ {
		mc.Set(benchkeys[i%benchcount], value)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var i = 0
		for pb.Next() {
			index := getIndex(i)
			i++
			mc.Get(benchkeys[index])
		}
	})
}

func BenchmarkFastCache_SetAndGet(b *testing.B) {
	cache, err := fastcache.NewCache(fastcache.GB, nil)
	if err != nil {
		b.Fatal(err)
	}
	mc := cache
	var value = []byte("1")
	for i := 0; i < benchcount; i++ {
		mc.Set(benchkeys[i%benchcount], value)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var i = 0
		for pb.Next() {
			index := getIndex(i)
			i++
			if index&7 == 0 {
				mc.Set(benchkeys[index], value)
			} else {
				mc.Get(benchkeys[index])
			}
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
			i++
			mc.SetWithTTL(benchkeys[index], 1, 1, time.Hour)
		}
	})
}

func TestBenchmarkTemp(t *testing.T) {
	var mc, _ = ristretto.NewCache(&ristretto.Config{
		NumCounters: capacity * sharding * 10, // number of keys to track frequency of (10M).
		MaxCost:     1 << 30,                  // maximum cost of cache (1GB).
		BufferItems: 64,                       // number of keys per Get buffer.
	})

	ok := mc.SetWithTTL("1", 1, 1, time.Hour)
	v, exists := mc.Get("1")
	t.Log("set ok: ", ok, " get value: ", v, " exists: ", exists)
}

func BenchmarkRistretto_Get(b *testing.B) {
	var mc, _ = ristretto.NewCache(&ristretto.Config{
		NumCounters: capacity * sharding * 10, // number of keys to track frequency of (10M).
		MaxCost:     1 << 30,                  // maximum cost of cache (1GB).
		BufferItems: 64,                       // number of keys per Get buffer.
	})
	for i := 0; i < benchcount; i++ {
		mc.SetWithTTL(benchkeys[i%benchcount], 1, 1, time.Hour)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var i = 0
		for pb.Next() {
			index := getIndex(i)
			i++
			mc.Get(benchkeys[index])
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
		mc.SetWithTTL(benchkeys[i%benchcount], 1, 1, time.Hour)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var i = 0
		for pb.Next() {
			index := getIndex(i)
			i++
			if index&7 == 0 {
				mc.SetWithTTL(benchkeys[index], 1, 1, time.Hour)
			} else {
				mc.Get(benchkeys[index])
			}
		}
	})
}

func BenchmarkTheine_Set(b *testing.B) {
	mc, _ := theine.NewBuilder[string, int](sharding * capacity).Build()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			index := getIndex(i)
			i++
			mc.SetWithTTL(benchkeys[index], 1, 1, time.Hour)
		}
	})
}

func BenchmarkTheine_Get(b *testing.B) {
	mc, _ := theine.NewBuilder[string, int](sharding * capacity).Build()
	for i := 0; i < benchcount; i++ {
		mc.SetWithTTL(benchkeys[i%benchcount], 1, 1, time.Hour)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			index := getIndex(i)
			i++
			mc.Get(benchkeys[index])
		}
	})
}

func BenchmarkTheine_SetAndGet(b *testing.B) {
	mc, _ := theine.NewBuilder[string, int](sharding * capacity).Build()
	for i := 0; i < benchcount; i++ {
		mc.SetWithTTL(benchkeys[i%benchcount], 1, 1, time.Hour)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			index := getIndex(i)
			i++
			if index&7 == 0 {
				mc.SetWithTTL(benchkeys[index], 1, 1, time.Hour)
			} else {
				mc.Get(benchkeys[index])
			}
		}
	})
}

// 测试LRU算法实现的正确性
func TestLRU_Impl(t *testing.T) {
	var f = func() {
		var count = 10000
		var capacity = 5000
		var mc = memorycache.New[string, int](
			memorycache.WithBucketNum(1),
			memorycache.WithBucketSize(capacity, capacity),
		)
		var cache, _ = lru.New[string, int](capacity)
		for i := 0; i < count; i++ {
			key := string(utils.AlphabetNumeric.Generate(16))
			val := utils.AlphabetNumeric.Intn(capacity)
			mc.Set(key, val, time.Hour)
			cache.Add(key, val)
		}

		keys := cache.Keys()
		assert.Equal(t, mc.Len(), capacity)
		assert.Equal(t, mc.Len(), cache.Len())
		assert.Equal(t, mc.Len(), len(keys))

		for _, key := range keys {
			v1, ok1 := mc.Get(key)
			v2, _ := cache.Peek(key)
			assert.True(t, ok1)
			assert.Equal(t, v1, v2)
		}
	}

	for i := 0; i < 10; i++ {
		f()
	}
}

func BenchmarkEmpty(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			i++
		}
	})
}
