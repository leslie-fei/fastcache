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
	MaxElementLen uint32
	// big data max length
	BigDataLen uint32
	// large data size block
	BigDataSize       uint32
	ShardPerAllocSize uint64
	// number of shards
	Shards uint32
}

func DefaultConfig() *Config {
	var defaultConfig = &Config{
		MemoryType:        GO,
		Shards:            uint32(runtime.NumCPU() * 4),
		BigDataLen:        4096,
		BigDataSize:       16 * KB,
		ShardPerAllocSize: 1 * MB,
		MaxElementLen:     100_000,
	}
	return defaultConfig
}
