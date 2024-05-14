package memlru

import "github.com/dolthub/maphash"

var hasher = maphash.NewHasher[string]()

var xxHashString = func(key string) uintptr {
	return uintptr(hasher.Hash(key))
}
