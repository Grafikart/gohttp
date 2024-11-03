package main

import (
	"io"
	"os"
	"path"
	"strings"
)

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
