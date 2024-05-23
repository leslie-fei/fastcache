module github.com/leslie-fei/fastcache/benchmark

go 1.21

toolchain go1.22.1

require (
	github.com/Yiling-J/theine-go v0.3.1
	github.com/allegro/bigcache/v3 v3.1.0
	github.com/cespare/xxhash/v2 v2.3.0
	github.com/dgraph-io/ristretto v0.1.1
	github.com/leslie-fei/fastcache v0.0.0-20240515083218-aa5c6353801b
)

require (
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/edsrzf/mmap-go v1.1.0 // indirect
	github.com/gammazero/deque v0.2.1 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b // indirect
	github.com/klauspost/cpuid/v2 v2.2.6 // indirect
	github.com/ncw/directio v1.0.5 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	golang.org/x/sys v0.20.0 // indirect
)

replace github.com/leslie-fei/fastcache v0.0.0-20240515083218-aa5c6353801b => ../
