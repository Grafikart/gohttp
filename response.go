package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

func responseHTTP1(r Request, w io.Writer) {
	w.Write([]byte("HTTP/1.1 200 OK\n"))
	w.Write([]byte(fmt.Sprintf("Content-Type: text/%s\n", getFileExtension(r.Path))))
	w.Write([]byte("\n"))
	readFileInto(r.Path, w)
}

func responseHTTP2(r Request, streamID uint32, framer *http2.Framer) {
	// Headers frame
	buff := bytes.NewBuffer([]byte{})
	encoder := hpack.NewEncoder(buff)
	encoder.WriteField(hpack.HeaderField{
		Name:  ":status",
		Value: "200",
	})
	encoder.WriteField(hpack.HeaderField{
		Name:  "content-type",
		Value: "text/" + getFileExtension(r.Path),
	})
	framer.WriteHeaders(http2.HeadersFrameParam{
		StreamID:      streamID,
		BlockFragment: buff.Bytes(),
		EndHeaders:    true,
	})

	// Data frame
	content, err := os.ReadFile(r.Path)
	if err != nil {
		framer.WriteData(streamID, true, []byte{})
		log.Printf("Cannot read file %s", r.Path)
		return
	}

	framer.WriteData(streamID, true, content)
}

func readFileInto(path string, w io.Writer) {
	content, err := os.ReadFile(path)
	if err != nil {
		w.Write([]byte("Cannot read file content"))
		return
	}
	w.Write(content)
}

func getFileExtension(filename string) string {
	fullExt := path.Ext(filename)
	ext := strings.TrimPrefix(fullExt, ".")

	return ext
}
