package fastcache

import "runtime"

type MemoryType int

const (
	GO   MemoryType = 1
	SHM             = 2
	MMAP            = 3
)

type Config struct {
	// memory type in GO SHM MMAP
	MemoryType MemoryType
	// shard memory key
	MemoryKey string
	// max element length in cache when set value over the length will evict oldest
	MaxElementLength uint64
	// number of shards
	Shards uint32
}

func DefaultConfig() *Config {
	return &Config{
		MemoryType: GO,
		Shards:     uint32(runtime.NumCPU()),
	}
}
