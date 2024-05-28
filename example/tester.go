package main

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/leslie-fei/fastcache"
)

const (
	keyMaxSize   = 32
	valueMaxSize = 1024
	numWorkers   = 1 // Number of worker goroutines
	//batchSize    = 100 // Number of operations per batch
	maxElements = 100000 // i cant make it more than 1 billion, will show no memory panic error
)

var (
	strSource = []byte("1234567890qwertyuiopasdfghjklzxcvbnm")
)

func getRandStr(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = strSource[rand.Intn(len(strSource))]
	}
	return string(b)
}

func getRandBytes(length int) []byte {
	b := make([]byte, length)
	rand.Read(b)
	return b
}

func worker(id int, cache *fastcache.Cache, requestCount *int64, mu *sync.Mutex) {
	for {
		keySize := rand.Intn(keyMaxSize) + 1
		valueSize := rand.Intn(valueMaxSize) + 1
		key := getRandStr(keySize)
		value := getRandBytes(valueSize)

		// Set
		err := (*cache).Set((key), value)
		if err != nil {
			log.Fatal(err)
		}

		// Get
		retrievedValue, err := (*cache).Get((key))
		if err != nil {
			log.Fatal(err)
		}
		//Verify
		if !equal(retrievedValue, value) {
			log.Printf("Mismatch: Key %s, Expected Value: %x, Retrieved Value: %x\n", key, value, retrievedValue)
		}

		mu.Lock()
		*requestCount += 2 // 1 SET + 1 GET
		mu.Unlock()
	}
}

func equal(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func runTester() {
	cache, err := fastcache.NewCache(24*fastcache.MB, &fastcache.Config{
		MemoryType: fastcache.GO,
		MemoryKey:  "./exampleSharedMemory.test",
		//Shards:        uint32(runtime.NumCPU() * 4),
		Shards:        1,
		MaxElementLen: maxElements,
	})

	if err != nil {
		panic(err)
	}

	var requestCount int64
	var mu sync.Mutex

	ticker := time.NewTicker(1 * time.Second)
	go func() {
		for range ticker.C {
			mu.Lock()
			fmt.Printf("Requests per second: %d\n", requestCount)
			requestCount = 0
			mu.Unlock()
		}
	}()

	var wg sync.WaitGroup

	// Start workers
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			worker(id, &cache, &requestCount, &mu)
		}(w)
	}

	wg.Wait()
	ticker.Stop()
}
