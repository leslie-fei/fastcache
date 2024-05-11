package memlru

type BlockFreeList struct {
	Len  uint32
	Head uint64
}

func (bl *BlockFreeList) First(mem *MemoryManager) *LinkedNode {
	if bl.Len == 0 {
		return nil
	}
	return (*LinkedNode)(mem.offset(bl.Head))
}
