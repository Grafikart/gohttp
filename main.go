package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/quic-go/quic-go"
)

func main() {
	// On récupère l'argument
	if len(os.Args) < 2 {
		log.Fatalf("Vous devez fournir le mode (http1, http2, ou http3)")
		fmt.Println("Utilisation:", os.Args[0], "<mode>")
		fmt.Println("Exemple:")
		fmt.Println("  go run . http1")
		fmt.Println("  go run . http2")
		fmt.Println("  go run . http3")
		os.Exit(1)
	}
	mode := os.Args[1]
	validModes := map[string]bool{
		"http1": true,
		"http2": true,
		"http3": true,
	}
	if !validModes[mode] {
		log.Fatalf("Erreur: Mode invalide '%s'. http1, http2, http3 accepté\n", mode)
	}

	fmt.Println("🖥️ Serveur démarré sur https://localhost")

	if mode == "http3" {
		config := loadTLSConfig(HTTP3)
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

		ln, err := tr.Listen(config, quicConf)
		go broadcastHTTP3(config)

		for {
			conn, err := ln.Accept(context.Background())
			if err != nil {
				log.Printf("impossible d'accepter la connexion, %v", err)
				continue
			}
			go handleHTTP3(conn)
		}
	} else {

		// On écoute les connexions TCP
		listener, err := net.Listen("tcp", ":443")
		if err != nil {
			log.Fatalf("Failed to create listener: %v\n", err)
		}
		defer listener.Close()

		for {
			protocol := HTTP1
			if mode == "http2" {
				protocol = HTTP2
			}
			// On accepte la connexion
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("Failed to accept connection: %v\n", err)
				continue
			}

			// Gestion du TLS
			tlsConn := tls.Server(conn, loadTLSConfig(protocol))
			err = tlsConn.Handshake()
			if err != nil {
				// Ignore error since we use local certificate
			}

			protocol = tlsConn.ConnectionState().NegotiatedProtocol

			switch protocol {
			case HTTP1:
				go handleHTTP1(tlsConn)
			case HTTP2:
				go handleHTTP2(tlsConn)
			}
		}
	}

}

func loadTLSConfig(protocol string) *tls.Config {
	// On charge les certificats
	cert, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
	if err != nil {
		log.Fatalf("Failed to load certificate: %v", err)
	}

	// Configuration TLS
	return &tls.Config{
		Certificates:       []tls.Certificate{cert},
		NextProtos:         []string{protocol},
		InsecureSkipVerify: true,
	}
}

func main2() {
}
