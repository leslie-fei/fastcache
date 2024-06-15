package fastcache

import (
	"encoding/binary"
	"encoding/json"
	"runtime"
)

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
	MaxElementLen uint64
	// 大数据块的最大数量, 超过这个数量将会触发淘汰, 并且这个数值将会用来初始化大数据块的定长Hashmap
	MaxBigDataLen uint64
	// 定义多少字节为大数据块
	BigDataSize uint32
	// 分片每次申请内存大小
	ShardPerAllocSize uint64
	// 分片数量
	Shards uint32
	Hasher HashFunc `json:"-"`
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

func mergeConfig(size int, c *Config) *Config {
	config := DefaultConfig()
	// 默认MaxElementLen通过设置的内存大小计算出来
	config.MaxElementLen = uint64(size / 512)
	// 默认是MaxElementLen的1/20
	config.MaxBigDataLen = config.MaxElementLen / 20
	if c != nil {
		config.MemoryKey = c.MemoryKey
		if c.MemoryType > 0 {
			config.MemoryType = c.MemoryType
		}
		if c.Shards > 0 {
			config.Shards = c.Shards
		}
		if c.MaxElementLen > 0 {
			config.MaxElementLen = c.MaxElementLen
		}
		if c.BigDataSize > 0 {
			config.BigDataSize = c.BigDataSize
		}
		if c.MaxBigDataLen > 0 {
			config.MaxBigDataLen = c.MaxBigDataLen
		}
		if c.Hasher != nil {
			xxHashBytes = c.Hasher
		}
	}
	return config
}

func getConfigHash(size int, config *Config) (uint64, error) {
	js, err := json.Marshal(config)
	if err != nil {
		return 0, err
	}
	js = append(js)
	js = binary.LittleEndian.AppendUint32(js, uint32(size))
	return xxHashBytes(js), nil
}
