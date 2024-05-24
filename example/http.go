package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"syscall"

	"github.com/leslie-fei/fastcache"
	"golang.org/x/sys/unix"
)

func runHTTPServer() {
	var lc = net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var opErr error
			if err := c.Control(func(fd uintptr) {
				opErr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
			}); err != nil {
				return err
			}
			return opErr
		},
	}

	cache, err := fastcache.NewCache(fastcache.GB, &fastcache.Config{
		MemoryType: fastcache.SHM,
		MemoryKey:  "/tmp/ExampleRunHttpServer",
	})
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/get", func(writer http.ResponseWriter, request *http.Request) {
		key := request.URL.Query().Get("k")
		v, err := cache.Get(key)
		handleResponse(writer, v, err)
	})

	http.HandleFunc("/set", func(writer http.ResponseWriter, request *http.Request) {
		key := request.URL.Query().Get("k")
		value := request.URL.Query().Get("v")
		err := cache.Set(key, []byte(value))
		handleResponse(writer, nil, err)
	})

	http.HandleFunc("/del", func(writer http.ResponseWriter, request *http.Request) {
		key := request.URL.Query().Get("k")
		err := cache.Delete(key)
		handleResponse(writer, nil, err)
	})

	addr := ":8080"
	l, err := lc.Listen(context.Background(), "tcp", addr)
	if err != nil {
		panic(err)
	}
	httpServer := &http.Server{}
	if err = httpServer.Serve(l); err != nil {
		panic(err)
	}

	fmt.Println("start http server success: ", addr)
}

type Result struct {
	Code  uint32 `json:"code"`
	Error string `json:"error"`
	Data  string `json:"data"`
}

func handleResponse(writer http.ResponseWriter, v []byte, err error) {
	var result = &Result{}
	if v != nil {
		result.Data = string(v)
	}
	if err != nil {
		result.Error = err.Error()
		result.Code = 1
	}
	r, err := json.Marshal(result)
	if err != nil {
		panic(err)
	}
	writer.Write(r)
}
