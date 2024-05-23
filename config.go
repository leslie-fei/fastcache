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
	// 支持存储的最大数量, 超过将会触发LRU
	MaxElementLen uint32
	// 大数据块的最大数量, 超过这个数量将会触发淘汰, 并且这个数值将会用来初始化大数据块的定长Hashmap
	MaxBigDataLen uint32
	// 定义多少字节为大数据块
	BigDataSize uint32
	// 分片每次申请内存大小
	ShardPerAllocSize uint64
	// 分片数量
	Shards uint32
}

func DefaultConfig() *Config {
	var defaultConfig = &Config{
		MemoryType:        GO,
		Shards:            uint32(runtime.NumCPU() * 4),
		BigDataSize:       16 * KB,
		ShardPerAllocSize: 1 * MB,
	}
	return defaultConfig
}
