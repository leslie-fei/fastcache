package main

import (
	//"encoding/hex"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"sync"
	"time"

	"github.com/leslie-fei/fastcache"
)

var mu sync.Mutex
var requestCount int64

const (
	numEntries     = 100000000000 // Total number of entries
	batchSize      = 117481       // Number of entries per batch
	keySize        = 1024
	valueSize      = 16000
	evictBatchSize = 1
	numWorkers     = 100 // Number of worker goroutines
)

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

func worker(id int, jobs <-chan int, wg *sync.WaitGroup, cache fastcache.Cache) {
	defer wg.Done()
	for _ = range jobs {
		key := generateRandomKey()
		value := generateRandomValue()
		err := cache.Set((key), []byte(value))
		if err != nil {
			panic(err)
		}
		retrievedValue, _ := cache.Peek((key))
		if string(retrievedValue) != value {

			log.Fatalf("Mismatch: Key %s, Expected Value: %s, Retrieved Value: %s\n",
				key, value, (retrievedValue))
		}

		mu.Lock()
		requestCount += 2 // 1 SET + 1 GET
		mu.Unlock()
		//              if i%10000 == 0 {
		//                      log.Printf("Worker %d: Inserted and verified %d entries\n", id, i)
		//              }
	}
}

func processBatch(start int, end int, cache fastcache.Cache) {
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
	//go cxsysinfodebug.LogMemStatsPeriodically(1*time.Second, &cxsysinfodebug.FileDescriptorTracker{})

	ticker := time.NewTicker(1 * time.Second)
	go func() {
		for range ticker.C {
			mu.Lock()
			fmt.Printf("Requests per second: %d\n", requestCount)
			requestCount = 0
			mu.Unlock()
		}
	}()

	go func() {
		err := http.ListenAndServe(":16060", nil)
		if err != nil {
			panic(err)
		}
	}()

	var maxMemoryMB int
	flag.IntVar(&maxMemoryMB, "m", 1024, "memory limit in MB (default 1024)")
	flag.Parse()

	cache, err := fastcache.NewCache(20*fastcache.GB, &fastcache.Config{
		MemoryType:    fastcache.MMAP,
		MemoryKey:     "/dev/shm/exampleSharedMemory",
		Shards:        uint32(runtime.NumCPU() * 10),
		MaxElementLen: batchSize * 10,
	})

	if err != nil {
		panic(err)
	}

	startTime := time.Now()

	for start := 0; start < numEntries; start += batchSize {
		end := start + batchSize
		if end > numEntries {
			end = numEntries
		}
		processBatch(start, end, cache)
	}

	elapsedTime := time.Since(startTime)
	fmt.Printf("Time taken to insert and verify %d entries: %s\n", numEntries, elapsedTime)
}
