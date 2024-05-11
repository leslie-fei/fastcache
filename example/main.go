package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"memlru"
	"memlru/shm"
)

func main() {
	var size int
	var key string

	// Example command: go run main.go --port 8080 --multicore=true
	flag.IntVar(&size, "m", 64, "memory limit unit MB default 64")
	flag.StringVar(&key, "k", "/shm/test", "attach share memory path")
	flag.Parse()

	size = size * memlru.MB

	mem := shm.NewMemory(key, uint64(size), true)
	if err := mem.Attach(); err != nil {
		panic(err)
	}
	defer func() {
		if err := mem.Detach(); err != nil {
			panic(err)
		}
	}()

	memMgr := memlru.NewMemoryManager(mem)
	if err := memMgr.Init(); err != nil {
		panic(err)
	}

	hashmap := memMgr.Hashmap()

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Available commands: set <key> <value>, get <key>, del <key>, list, exit")

	for scanner.Scan() {
		input := scanner.Text()
		parts := strings.SplitN(input, " ", 3)

		switch parts[0] {
		case "exit":
			break
		case "set":
			if len(parts) != 3 {
				fmt.Println("Usage: set <key> <value>")
				continue
			}
			hashmap.Set(parts[1], []byte(parts[2]))
			fmt.Println("Set completed.")
		case "get":
			if len(parts) != 2 {
				fmt.Println("Usage: get <key>")
				continue
			}
			result, err := hashmap.Get(parts[1])
			if err != nil && !errors.Is(err, memlru.ErrNotFound) {
				panic(err)
			}

			if errors.Is(err, memlru.ErrNotFound) {
				fmt.Println("key not found")
				continue
			}

			fmt.Printf("key: %s value: %s\n", parts[1], result)
		case "del":
			if len(parts) != 2 {
				fmt.Println("Usage: del <key>")
				continue
			}
			hashmap.Del(parts[1])
			fmt.Println("Del completed.")
		default:
			fmt.Println("Unknown command. Try: set, get, del, list, or exit.")
		}
		fmt.Println("Enter a command (type 'exit' to quit):")
	}
}
