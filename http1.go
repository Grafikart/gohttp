package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"strconv"

	"net"
	"strings"
	"time"

	"github.com/fatih/color"
)

const HTTP1 = "http/1.1"

type Request struct {
	Path     string
	Method   string
	Protocol string
	Headers  map[string]string
	Body     string
}

// Les requêtes HTTP1 ne contiennent qu'une requête par connection
// La requête est présentée sous forme de texte contenant l'ensemble des informations
func handleHTTP1(conn net.Conn) {
	defer conn.Close()
	r, err := NewHTTP1Request(conn)
	if err != nil {
		// Silence les erreurs de certificat
		if strings.Contains(err.Error(), "unknown certificate") {
			return
		}
		log.Printf("Error handling request %v", err.Error())
		return
	}
	respondHTTP1(r, conn)
	fmt.Printf("x\n\n")
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
*
* firstname=John
* ```
**/
func NewHTTP1Request(c net.Conn) (*Request, error) {
	// Pour éviter le blocage, on ne lit plus au bout de 2 secondes
	c.SetReadDeadline(time.Now().Add(2 * time.Second))

	r := bufio.NewReader(c)

	// On lit la première ligne
	method, path, protocol, err := readRequestLine(r)
	if err != nil {
		return nil, err
	}
	fmt.Printf("⦿\n")
	printLine(fmt.Sprintf("%s %s %s", method, path, protocol), true)

	req := &Request{
		Method:   method,
		Path:     resolvePath(path),
		Protocol: protocol,
		Headers:  make(map[string]string),
	}

	// On lit les en têtes
	for {
		name, value := readHeaderLine(r)
		printHeader(name, value, true)
		if name == "" {
			break
		}
		req.Headers[name] = value
	}

	// On lit le body
	lengthHeader, ok := req.Headers["Content-Length"]
	if ok {
		contentLength, err := strconv.Atoi(lengthHeader)
		if contentLength > 0 && err == nil {
			body, err := r.Peek(contentLength)
			if err == nil {
				req.Body = string(body)
				printLine(req.Body, true)
			}
		}
	}

	// On lit les en-têtes
	return req, nil
}

func readRequestLine(r *bufio.Reader) (string, string, string, error) {
	l, err := r.ReadString('\n')
	if err != nil {
		return "", "", "", fmt.Errorf("impossible de lire la première ligne, %s\n", err.Error())
	}
	l = strings.TrimSpace(l)
	parts := strings.Split(l, " ")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf(
			"impossible de lire la première ligne, 3 parties attendues, %v\n", l)
	}
	return parts[0], parts[1], parts[2], nil
}

func readHeaderLine(r *bufio.Reader) (string, string) {
	l, err := r.ReadString('\n')
	if err != nil {
		return "", ""
	}
	l = strings.TrimSpace(l)
	parts := strings.Split(l, ":")
	if len(parts) >= 2 {
		return parts[0],
			strings.TrimSpace(strings.Join(parts[1:], ":"))
	}
	return "", ""
}

func printLine(s string, in bool) {
	dirColor(in).Printf("| %s\n", s)
}

func printHeader(name string, value string, in bool) {
	if name == "" {
		printLine(name, in)
		return
	}
	if len(name)+len(value) > 50 {
		value = value[0:(50-len(name))] + "..."
	}
	fmt.Printf(
		"%s: %s\n",
		dirColor(in).Sprintf("| "+name),
		value,
	)
}

// Trouve la couleur à utiliser (bleu pour la lecture, vert pour l'écriture)
func dirColor(in bool) *color.Color {
	if in == false {
		return green
	}
	return blue
}

// Trouve le fichier à charger en fonction du chemin demandé
func resolvePath(path string) string {
	if strings.HasSuffix(path, "/") {
		path = path + "index.html"
	}
	return strings.Trim(path, "/")
}

func respondHTTP1(r *Request, w io.Writer) {
	w.Write([]byte("HTTP/1.1 200 OK\n"))
	format := fmt.Sprintf("text/%s", getFileExtension(r.Path))
	w.Write([]byte("Content-Type: " + format + "\n"))
	w.Write([]byte("\n"))
	readFileInto("public/"+r.Path, w)

	printLine("HTTP/1.1 200 OK", false)
	printHeader("Content-Type", format, false)
	printLine("", false)
	printLine("...", false)
}
