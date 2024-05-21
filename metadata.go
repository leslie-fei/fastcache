package fastcache

type Metadata struct {
	Magic          uint64
	TotalSize      uint64
	Used           uint64
	Shards         uint32
	LockerOffset   uint64 // 全局锁
	BigShardOffset uint64
	ShardsOffset   uint64
}

func (m *Metadata) Reset() {
	*m = Metadata{}
}
