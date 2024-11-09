package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"

	"github.com/k0kubun/pp/v3"
	"github.com/quic-go/quic-go"
)

// localhost;h3=":443";h3-29=":443"

func main() {
	fmt.Println("🖥️ listening on https://localhost")
	config := loadTLSConfig()
	udpConn, err := net.ListenUDP("udp4", &net.UDPAddr{Port: 443, IP: net.IPv4(0, 0, 0, 0)})
	if err != nil {
		log.Fatalf("impossible de créer la connection udb sur le port 443, %v", err)
	}
	tr := quic.Transport{
		Conn: udpConn,
	}
	quicConf := &quic.Config{
		Versions: []quic.Version{quic.Version2, quic.Version1},
	}

	ln, err := tr.Listen(loadTLSConfig(), quicConf)
	go broadcastHTTP3(config)

	for {
		conn, err := ln.Accept(context.Background())
		pp.Println("Accepting quic localhost:443")
		if err != nil {
			log.Printf("impossible d'accepter la connexion, %v", err)
			continue
		}
		go handleHTTP3(conn)
	}
}

func loadTLSConfig() *tls.Config {
	// On charge les certificats
	cert, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
	if err != nil {
		log.Fatalf("Failed to load certificate: %v", err)
	}

	// Configuration TLS
	return &tls.Config{
		Certificates:       []tls.Certificate{cert},
		NextProtos:         []string{HTTP3},
		InsecureSkipVerify: true,
	}
}

func main2() {

	// On écoute les connexions TCP
	listener, err := net.Listen("tcp", ":443")
	if err != nil {
		log.Fatalf("Failed to create listener: %v\n", err)
	}
	defer listener.Close()

	fmt.Println("🖥️ listening on https://localhost:443")

	for {
		// On accepte la connexion
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v\n", err)
			continue
		}

		// Gestion du TLS
		tlsConn := tls.Server(conn, loadTLSConfig())
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
