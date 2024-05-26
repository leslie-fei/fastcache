package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"github.com/leslie-fei/fastcache"
)

const (
	numEntries = 1 << 20 // Total number of entries (example: 1 million)
	keySize    = 32
	valueSize  = 1024
	numWorkers = 100 // Number of worker goroutines
)

var (
	benchKeys = make([]string, numEntries)
	benchVals = make([][]byte, numEntries)
	strSource = []byte("1234567890qwertyuiopasdfghjklzxcvbnm")
)

func init() {
	for i := 0; i < numEntries; i++ {
		benchKeys[i] = getRandStr(keySize)
		benchVals[i] = make([]byte, valueSize)
		rand.Read(benchVals[i])
	}
}

func getRandStr(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = strSource[rand.Intn(len(strSource))]
	}
	return string(b)
}

func getIndex(i int) int {
	return i & (numEntries - 1)
}

func worker(id int, jobs <-chan int, wg *sync.WaitGroup, cache *fastcache.Cache, requestCount *int64, mu *sync.Mutex) {
	defer wg.Done()
	for i := range jobs {
		index := getIndex(i)
		key := benchKeys[index]
		value := benchVals[index]

		// Set
		(*cache).Set((key), value)

		// Get
		retrievedValue, _ := (*cache).Get((key))

		// Verify
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

func processBatch(cache *fastcache.Cache, requestCount *int64, mu *sync.Mutex) {
	var wg sync.WaitGroup
	jobs := make(chan int, numEntries)

	// Start workers
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go worker(w, jobs, &wg, cache, requestCount, mu)
	}

	// Send jobs
	for i := 0; i < numEntries; i++ {
		jobs <- i
	}
	close(jobs)

	// Wait for all workers to finish
	wg.Wait()
}

func runTester() {
	var maxMemoryMB int
	flag.IntVar(&maxMemoryMB, "m", 1024, "memory limit in MB (default 1024)")
	flag.Parse()

	cache, err := fastcache.NewCache(5*fastcache.GB, &fastcache.Config{
		MemoryType:    fastcache.MMAP,
		MemoryKey:     "/tmp/exampleSharedMemory",
		Shards:        uint32(runtime.NumCPU() * 4),
		MaxElementLen: numEntries,
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

	startTime := time.Now()

	processBatch(&cache, &requestCount, &mu)

	ticker.Stop()
	elapsedTime := time.Since(startTime)
	fmt.Printf("Time taken to insert and verify %d entries: %s\n", numEntries, elapsedTime)
}
