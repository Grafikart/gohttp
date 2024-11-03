package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
)

func main() {
	// On charge les certificats
	cert, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
	if err != nil {
		log.Fatalf("Failed to load certificate: %v", err)
	}

	// Configuration TLS
	config := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		NextProtos:         []string{HTTP2, HTTP1},
		InsecureSkipVerify: true,
	}

	// On écoute les connexions TCP
	listener, err := net.Listen("tcp", ":8000")
	if err != nil {
		log.Fatalf("Failed to create listener: %v\n", err)
	}
	defer listener.Close()

	fmt.Println("🖥️ listening on https://localhost:8000")

	for {
		// On accepte la connexion
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v\n", err)
			continue
		}

		// Gestion du TLS
		tlsConn := tls.Server(conn, config)
		err = tlsConn.Handshake()
		if err != nil {
			// Ignore error since we use local certificate
		}

		protocol := tlsConn.ConnectionState().NegotiatedProtocol

		switch protocol {
		case HTTP1:
			go handleHTTP1(tlsConn)
		case HTTP2:
			go handleHTTP2(tlsConn)
		}
	}
}
