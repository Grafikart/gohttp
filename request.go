package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

const HTTP1 = "http/1.1"
const HTTP2 = "h2"

type Request struct {
	Path     string
	Method   string
	Protocol string
	Headers  map[string]string
}

/**
* Construit une requête à partir d'un message HTTP1
*
* ## Exemple de message
*
* ```
* GET / HTTP/1.1
* Host: www.example.com
* User-Agent: Mozilla/5.0 ...
* Accept: text/html
* Accept-Encoding: gzip, deflate, br
* Connection: keep-alive
* ```
**/
func NewHTTP1Request(conn io.Reader) (Request, error) {
	r := bufio.NewReader(conn)

	// La première ligne contient le type de la requête "METHOD PATH PROTOCOL"
	l := readLine(r)
	parts := strings.Split(l, " ")
	if len(parts) < 3 {
		return Request{}, fmt.Errorf("Cannot generate request from %v\n", l)
	}
	method := parts[0]
	path := parts[1]
	protocol := parts[2]
	headers := make(map[string]string)
	if strings.HasSuffix(path, "/") {
		path = path + "index.html"
	}
	path = strings.Trim(path, "/")

	// Puis les en têtes
	for {
		l = readLine(r)
		if l == "" {
			break
		}
		parts := strings.Split(l, ":")
		if len(parts) < 2 {
			return Request{}, fmt.Errorf("Unexpected header %s\n", l)
		}
		headers[strings.ToLower(parts[0])] = strings.TrimSpace(parts[1])
	}

	return Request{
		Path:     path,
		Method:   method,
		Protocol: protocol,
		Headers:  headers,
	}, nil
}

var decoder = hpack.NewDecoder(2048, nil)

func NewHTTP2Request(f *http2.HeadersFrame) (Request, error) {
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

	return Request{
		Path:     path,
		Method:   "GET",
		Protocol: "h2",
		Headers:  headers,
	}, nil
}

func readLine(r *bufio.Reader) string {
	l, err := r.ReadString('\n')
	if err != nil && !strings.Contains(err.Error(), "unknown certificate") {
		fmt.Printf("Cannot read line, %v", err.Error())
	}
	return strings.TrimSpace(l)
}
