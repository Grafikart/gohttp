package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"strings"

	"github.com/k0kubun/pp/v3"
)

func main() {
	listener, err := net.Listen("tcp", "127.0.0.1:8000")

	if err != nil {
		pp.Printf("Cannot listen to localhost:8000, %v", err.Error())
		os.Exit(1)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			pp.Printf("Cannot accept connection, %v", err.Error())
		} else {
			go handleRequest(conn)
		}
	}
}

func handleRequest(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	// Read first line that should contain requested element
	parts := strings.Split(readLine(reader), " ")
	method := parts[0]
	path := parts[1]
	_ = parts[2]

	if strings.HasSuffix(path, "/") {
		path = fmt.Sprintf("%sindex.html", path)
	}

	path = strings.Trim(path, "/")
	fmt.Printf("> Req : %s %s\n", method, path)

	conn.Write([]byte("HTTP/1.1 200 OK\n"))
	conn.Write([]byte(fmt.Sprintf("Content-Type: text/%s\n", getFileExtension(path))))
	conn.Write([]byte("\n"))
	readFileInto(path, conn)
}

func readFileInto(path string, w io.Writer) {
	content, err := os.ReadFile(path)
	if err != nil {
		w.Write([]byte("Cannot read file content"))
		return
	}
	w.Write(content)
}

func readLine(r *bufio.Reader) string {
	l, err := r.ReadString('\n')
	if err != nil {
		pp.Printf("Cannot read line, %v", err.Error())
	}
	return strings.TrimSpace(l)
}

func readEmptyLine(r *bufio.Reader) {
	l := readLine(r)
	if l != "" {
		pp.Println("Expected empty line there")
	}
}

func getFileExtension(filename string) string {
	// Get the full extension (including the dot)
	fullExt := path.Ext(filename)

	// Remove the leading dot
	ext := strings.TrimPrefix(fullExt, ".")

	return ext
}
