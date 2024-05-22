module github.com/leslie-fei/fastcache/benchmark

go 1.21

toolchain go1.22.1

require (
	github.com/Yiling-J/theine-go v0.3.1
	github.com/dgraph-io/ristretto v0.1.1
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/leslie-fei/fastcache v0.0.0-20240515083218-aa5c6353801b
	github.com/stretchr/testify v1.9.0
)

require (
	github.com/allegro/bigcache/v3 v3.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dolthub/maphash v0.1.0 // indirect
	github.com/dolthub/swiss v0.2.1 // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/edsrzf/mmap-go v1.1.0 // indirect
	github.com/gammazero/deque v0.2.1 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b // indirect
	github.com/klauspost/cpuid/v2 v2.2.6 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lxzan/dao v1.1.6 // indirect
	github.com/ncw/directio v1.0.5 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	golang.org/x/sys v0.20.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/leslie-fei/fastcache v0.0.0-20240515083218-aa5c6353801b => ../
