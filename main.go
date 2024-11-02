package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/k0kubun/pp/v3"
	"golang.org/x/net/http2"
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
		NextProtos:         []string{HTTP1},
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

// Les requêtes HTTP2 fonctionnent avec un système de frames
func handleHTTP2(conn net.Conn) {
	defer conn.Close()
	_, err := readBytes(conn, 24)

	if err != nil {
		log.Printf("Cannot read preface %v\n", err)
	}
	pp.Printf("\n\nx-- Preface -->")

	framer := http2.NewFramer(conn, conn)

	/*
		// Send initial settings to the client
		if err := framer.WriteSettings(
			http2.Setting{ID: http2.SettingMaxConcurrentStreams, Val: 100},
			http2.Setting{ID: http2.SettingInitialWindowSize, Val: 65535},
		); err != nil {
			log.Printf("Failed to send settings frame: %v\n", err)
			return
		}
	*/

	for {
		frame, err := framer.ReadFrame()
		if err != nil {
			log.Printf("Cannot read frame %s", err.Error())
		}
		switch f := frame.(type) {
		case *http2.SettingsFrame:
			if err := framer.WriteSettingsAck(); err != nil {
				log.Printf("Failed to write settings ACK: %v\n", err)
				return
			}
			UNUSED(f)
		case *http2.WindowUpdateFrame:
			pp.Printf("-- Window Update -->")
			UNUSED(f)
		case *http2.HeadersFrame:
			streamID := f.StreamID
			r, _ := NewHTTP2Request(f)
			pp.Printf("-- HeadersFrame %v %v %v -->", streamID, r.Path, f.Flags.Has(http2.FlagHeadersEndStream))
			// On répond
			if r.Path == "" {
				// Comprendre pourquoi on a des en tête vide :(
				framer.WriteHeaders(http2.HeadersFrameParam{
					StreamID:   streamID,
					EndHeaders: true,
					EndStream:  true,
				})
			} else {
				responseHTTP2(r, streamID, framer)
			}
		case *http2.RSTStreamFrame:
			log.Printf("Stream %v terminated", f.StreamID)
		case *http2.GoAwayFrame:
			log.Printf("GoAway", f.StreamID)
			return
		default:
			log.Printf("Received frame of type %T", f)
			return
		}
	}

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
