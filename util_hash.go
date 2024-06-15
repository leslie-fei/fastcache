package fastcache

import (
	"github.com/cespare/xxhash/v2"
)

var xxHashBytes = func(key []byte) uint64 {
	return xxhash.Sum64(key)
}
