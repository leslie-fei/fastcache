package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	cx "github.com/cloudxaas/gocx"
	cxsysinfodebug "github.com/cloudxaas/gosysinfo/debug"
	"github.com/leslie-fei/fastcache"
)

var mu sync.Mutex
var requestCount int64

const (
	numEntries     = 100000000000 // Total number of entries
	batchSize      = 976562       // Number of entries per batch
	keySize        = 1024
	valueSize      = 1024
	evictBatchSize = 1
	numWorkers     = 100 // Number of worker goroutines
)

func generateRandomBytes(dst *[1024]byte) {
	_, err := rand.Read(dst[:])
	if err != nil {
		log.Fatalf("Error generating random bytes: %s", err)
	}
}

func generateRandomKey(dst *[2048]byte) {
	var bytes [1024]byte
	generateRandomBytes(&bytes)
	hex.Encode(dst[:], bytes[:])
}

func generateRandomValue(dst *[2048]byte) int {
	length := 32 + rand.Intn(993)
	var bytes [1024]byte
	generateRandomBytes(&bytes)
	hex.Encode(dst[:], bytes[:length])
	return length * 2 // Hex encoding doubles the length
}

/*
func generateRandomBytes(n int) []byte {
        b := make([]byte, n)
        _, err := rand.Read(b)
        if err != nil {
                log.Fatalf("Error generating random bytes: %s", err)
        }
        return b
}

func generateRandomKey() string {
        return fmt.Sprintf("%x", generateRandomBytes(1024))
}

func generateRandomValue() string {
        length := 32 + rand.Intn(993)
        return fmt.Sprintf("%x", generateRandomBytes(length))
}
*/

var bufferPool = &sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 1024))
	},
}

func worker(id int, jobs <-chan int, wg *sync.WaitGroup, cache *fastcache.Cache) {
	defer wg.Done()

	var key [2048]byte
	var value [2048]byte
	for _ = range jobs {
		generateRandomKey(&key)
		length := generateRandomValue(&value)

		keyStr := cx.B2s(key[:2048])
		valueStr := cx.B2s(value[:length])

		err := (*cache).Set(keyStr, value[:length])
		if err != nil {
			panic(err)
		}

		buffer := bufferPool.Get().(*bytes.Buffer)
		buffer.Reset()
		_ = (*cache).PeekWithBuffer(keyStr, buffer)
		retrievedValue := buffer.Bytes()
		if string(retrievedValue) != valueStr {
			log.Fatalf("Mismatch: Key %s, Expected Value: %s, Retrieved Value: %s\n",
				keyStr, valueStr, string(retrievedValue))
		}
		bufferPool.Put(buffer)

		mu.Lock()
		requestCount += 2 // 1 SET + 1 GET
		mu.Unlock()
	}
}

func processBatch(start int, end int, cache *fastcache.Cache) {
	var wg sync.WaitGroup
	jobs := make(chan int, end-start)

	// Start workers
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go worker(w, jobs, &wg, cache)
	}

	// Send jobs
	for i := start; i < end; i++ {
		jobs <- i
	}
	close(jobs)

	// Wait for all workers to finish
	wg.Wait()
}

func main() {
	go cxsysinfodebug.LogMemStatsPeriodically(1*time.Second, &cxsysinfodebug.FileDescriptorTracker{})

	ticker := time.NewTicker(1 * time.Second)
	go func() {
		for range ticker.C {
			mu.Lock()
			fmt.Printf("Requests per second: %d\n", requestCount)
			requestCount = 0
			mu.Unlock()
		}
	}()

	var maxMemoryMB int
	flag.IntVar(&maxMemoryMB, "m", 1024, "memory limit in MB (default 1024)")
	flag.Parse()

	cache, _ := fastcache.NewCache(20*fastcache.GB, &fastcache.Config{
		MemoryType:    fastcache.MMAP,
		MemoryKey:     "/dev/shm/exampleSharedMemory",
		Shards:        128,
		MaxElementLen: batchSize * 10,
	})

	startTime := time.Now()

	for start := 0; start < numEntries; start += batchSize {
		end := start + batchSize
		if end > numEntries {
			end = numEntries
		}
		processBatch(start, end, &cache)
	}

	elapsedTime := time.Since(startTime)
	fmt.Printf("Time taken to insert and verify %d entries: %s\n", numEntries, elapsedTime)
}
