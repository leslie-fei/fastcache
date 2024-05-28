package fastcache

import "unsafe"

var sizeOfMetadata = unsafe.Sizeof(metadata{})

type metadata struct {
	Magic          uint64
	Hash           uint64
	TotalSize      uint64
	Used           uint64
	LockerOffset   uint64
	ShardArrOffset uint64
}

func (m *metadata) reset() {
	m.Magic = 0
	m.Hash = 0
	m.TotalSize = 0
	m.Used = 0
	m.ShardArrOffset = 0
}
