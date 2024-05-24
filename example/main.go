package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/leslie-fei/fastcache"
)

func main() {
	// 启动一个httpServer用来做测试
	runHTTPServer()
}

func runConsole() {
	var size int
	var key string

	// Example command: go run main.go --port 8080 --multicore=true
	flag.IntVar(&size, "m", 64, "memory limit unit MB default 64")
	flag.StringVar(&key, "k", "/tmp/TestSharedMemory", "attach share memory path")
	flag.Parse()

	size = size * fastcache.MB

	cache, err := fastcache.NewCache(128*fastcache.MB, &fastcache.Config{
		MemoryType: fastcache.SHM,
		MemoryKey:  "/tmp/ExampleSharedMemory",
	})
	if err != nil {
		panic(err)
	}

	defer cache.Close()

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Available commands: set <key> <value>, get <key>, del <key>, exit")

	for scanner.Scan() {
		input := scanner.Text()
		parts := strings.SplitN(input, " ", 3)

		switch parts[0] {
		case "exit":
			return
		case "set":
			if len(parts) != 3 {
				fmt.Println("Usage: set <key> <value>")
				continue
			}
			cache.Set(parts[1], []byte(parts[2]))
			fmt.Println("Set completed.")
		case "get":
			if len(parts) != 2 {
				fmt.Println("Usage: get <key>")
				continue
			}
			result, err := cache.Get(parts[1])
			if err != nil && !errors.Is(err, fastcache.ErrNotFound) {
				panic(err)
			}

			if errors.Is(err, fastcache.ErrNotFound) {
				fmt.Println("key not found")
				continue
			}

			fmt.Printf("key: %s value: %s\n", parts[1], result)
		case "del":
			if len(parts) != 2 {
				fmt.Println("Usage: del <key>")
				continue
			}
			if err := cache.Delete(parts[1]); err != nil {
				fmt.Println("delete error: ", err)
				continue
			}
			fmt.Println("Del completed.")
		default:
			fmt.Println("Unknown command. Try: set, get, del or exit.")
		}
	}
}
