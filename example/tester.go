package main

import (
	//"encoding/hex"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/leslie-fei/fastcache"
)

const (
	numEntries     = 100000000000 // Total number of entries
	batchSize      = 1000         // Number of entries per batch
	keySize        = 32
	valueSize      = 1024
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
	return fmt.Sprintf("%x", generateRandomBytes(32))
}

func generateRandomValue() string {
	length := 32 + rand.Intn(993)
	return fmt.Sprintf("%x", generateRandomBytes(length))
}

func worker(id int, jobs <-chan int, wg *sync.WaitGroup, cache *fastcache.Cache) {
	defer wg.Done()
	for i := range jobs {
		key := generateRandomKey()
		value := generateRandomValue()
		err := (*cache).Set((key), []byte(value))
		if err != nil {
			panic(err)
		}

		retrievedValue, _ := (*cache).Peek((key))

		if string(retrievedValue) != value {

			log.Printf("Mismatch: Key %s, Expected Value: %s, Retrieved Value: %s\n",
				key, value, (retrievedValue))
		}

		if i%10000 == 0 {
			log.Printf("Worker %d: Inserted and verified %d entries\n", id, i)
		}
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
	var maxMemoryMB int
	flag.IntVar(&maxMemoryMB, "m", 1024, "memory limit in MB (default 1024)")
	flag.Parse()

	//runtime.GOMAXPROCS(1)

	cache, _ := fastcache.NewCache(5*fastcache.GB, &fastcache.Config{
		MemoryType:    fastcache.MMAP,
		MemoryKey:     "/tmp/exampleSharedMemory",
		Shards:        128,
		MaxElementLen: batchSize * 10,
	})

	//    cache := fastcache.New(maxMemoryMB * fastcache.MB)
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
