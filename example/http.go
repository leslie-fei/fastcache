package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/leslie-fei/fastcache"
)

func runHTTPServer() {
	cache, err := fastcache.NewCache(fastcache.GB, nil)
	if err != nil {
		panic(err)
	}
	srv := http.NewServeMux()
	srv.Handle("/get", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		key := request.URL.Query().Get("k")
		v, err := cache.Get(key)
		handleResponse(writer, v, err)
	}))
	srv.Handle("/set", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		key := request.URL.Query().Get("k")
		value := request.URL.Query().Get("v")
		err := cache.Set(key, []byte(value))
		handleResponse(writer, nil, err)
	}))
	srv.Handle("/del", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		key := request.URL.Query().Get("k")
		err := cache.Delete(key)
		handleResponse(writer, nil, err)
	}))

	addr := ":8080"
	if err = http.ListenAndServe(addr, srv); err != nil {
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
