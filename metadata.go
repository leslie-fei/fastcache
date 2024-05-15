package fastcache

type Metadata struct {
	GlobalLocker             Locker // 全局锁
	Magic                    uint64
	TotalSize                uint64
	Used                     uint64
	LRUListOffset            uint64
	HashMapOffset            uint64
	BlockFreeContainerOffset uint64
	Lockers                  [192]Locker // 分段锁
}

func (m *Metadata) Reset() {
	m.Magic = 0
	m.TotalSize = 0
	m.Used = 0
	m.HashMapOffset = 0
	m.BlockFreeContainerOffset = 0
	for i := 0; i < len(m.Lockers); i++ {
		locker := &m.Lockers[i]
		locker.Reset()
	}
}
