package fastcache

type Metadata struct {
	GlobalLocker Locker // 全局锁
	Magic        uint64
	TotalSize    uint64
	Used         uint64
	Shards       uint32
}

func (m *Metadata) Reset() {
	m.Magic = 0
	m.TotalSize = 0
	m.Used = 0
}
