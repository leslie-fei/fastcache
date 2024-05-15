package fastcache

import "github.com/dolthub/maphash"

var hasher = maphash.NewHasher[string]()

var xxHashString = func(key string) uint64 {
	return hasher.Hash(key)
}
