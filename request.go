package main

import (
	"strings"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

const HTTP1 = "http/1.1"
const HTTP2 = "h2"

var decoder = hpack.NewDecoder(2048, nil)

func NewHTTP2Request(f *http2.HeadersFrame) (*Request, error) {
	hf, _ := decoder.DecodeFull(f.HeaderBlockFragment())
	path := ""
	headers := make(map[string]string)
	for _, h := range hf {
		if h.Name == ":path" {
			path = h.Value
		}
		headers[strings.ToLower(h.Name)] = strings.TrimSpace(h.Value)
	}
	if strings.HasSuffix(path, "/") {
		path = path + "index.html"
	}
	path = strings.Trim(path, "/")

	return &Request{
		Path:     path,
		Method:   "GET",
		Protocol: "h2",
		Headers:  headers,
	}, nil
}
