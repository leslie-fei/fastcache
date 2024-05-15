package fastcache

import (
	"errors"
	"math"
)

var ErrIndexOutOfRange = errors.New("index out of range")

type BlockFreeContainer struct {
	FreeLists [25]BlockFreeList
}

func (b *BlockFreeContainer) Init() {
	size := 1
	for i := 0; i < len(b.FreeLists); i++ {
		freeList := &b.FreeLists[i]
		freeList.Size = uint64(size)
		freeList.Index = uint8(i)
		size *= 2
	}
}

func (b *BlockFreeContainer) Get(dataSize uint64) (*BlockFreeList, error) {
	if dataSize == 0 {
		return nil, errors.New("data size is zero")
	}
	v := math.Log2(float64(dataSize))
	idx := int(math.Ceil(v))
	if idx > len(b.FreeLists)-1 {
		return nil, ErrIndexOutOfRange
	}
	return &b.FreeLists[idx], nil
}

func (b *BlockFreeContainer) GetIndex(idx uint8) *BlockFreeList {
	return &b.FreeLists[idx]
}

func (b *BlockFreeContainer) MaxSize() uint64 {
	return b.FreeLists[len(b.FreeLists)-1].Size
}

type BlockFreeList struct {
	Head  uint64 // head of data DataNode
	Len   uint32 // data len
	Size  uint64 // block bytes size
	Index uint8
}

func (bl *BlockFreeList) First(mem *MemoryManager) *DataNode {
	if bl.Len == 0 {
		return nil
	}
	return (*DataNode)(mem.offset(bl.Head))
}
