package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/k0kubun/pp/v3"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

const HTTP2 = "h2"

var decoder = hpack.NewDecoder(2048, nil)

func handleHTTP2(conn net.Conn) {
	defer conn.Close()
	fmt.Printf("⦿\n")
	defer fmt.Printf("x\n")

	// On lit la préface
	preface, err := readBytes(conn, 24)
	if err != nil {
		log.Printf("impossible de lire la préface %v", err.Error())
		return
	}
	dirColor(true).Printf("+- Preface : %q\n|\n", string(preface))

	// Permet de dupliquer le writer pour pouvoir écouter les frames renvoyées au client
	pr, pw := io.Pipe()
	multiWriter := io.MultiWriter(pw, conn)
	framer := http2.NewFramer(multiWriter, conn)

	go frameListener(pr, false)

	// Ecoute les frames entrantes
	requests := make(map[uint32]*Request)
	for {
		frame, err := framer.ReadFrame()
		if err == io.EOF {
			printKeyValue("EOF", true, true)
			return
		}
		if err != nil {
			log.Printf("Impossible de lire la frame %s", err.Error())
			return
		}
		printFrame(frame, true)
		switch f := frame.(type) {
		case *http2.SettingsFrame:
			err := framer.WriteSettingsAck()
			if err != nil {
				log.Printf("Impossible d'envoyer l'acceptation des Settings")
				return
			}
		case *http2.WindowUpdateFrame:
		case *http2.HeadersFrame:
			if !f.Flags.Has(http2.FlagHeadersEndHeaders) {
				continue
			}

			r, err := NewHTTP2Request(f)
			if err != nil {
				log.Printf("Impossible d'interpréter les en têtes")
				return
			}

			// On peut commencer à répondre
			requests[f.StreamID] = r

			if f.Flags.Has(http2.FlagHeadersEndStream) {
				respondHTTP2(r, f.StreamID, framer)
				delete(requests, f.StreamID)
			}

		case *http2.DataFrame:
			if f.Flags.Has(http2.FlagDataEndStream) {
				respondHTTP2(requests[f.StreamID], f.StreamID, framer)
				delete(requests, f.StreamID)
			}
		case *http2.GoAwayFrame:
			return
		default:
			fmt.Printf("Unknown type %T\n", f)
		}

	}

}

func frameListener(r io.Reader, in bool) {
	framer := http2.NewFramer(nil, r)

	for {
		frame, err := framer.ReadFrame()
		if err != nil {
			return
		}
		printFrame(frame, in)
	}
}

var printMu sync.Mutex

// Ecoute les frame et affiche dans le terminal
func printFrame(f http2.Frame, in bool) {
	printMu.Lock()
	defer printMu.Unlock()
	switch f := f.(type) {
	case *http2.SettingsFrame:
		printSettingFrame(f, in)
	case *http2.WindowUpdateFrame:
		printUpdateFrame(f, in)
	case *http2.HeadersFrame:
		printHeadersFrame(f, in)
	case *http2.DataFrame:
		printDataFrame(f, in)
	}
}

func printSettingFrame(f *http2.SettingsFrame, in bool) {
	color := dirColor(in)
	defer color.Printf("|\n")
	color.Printf("+- SETTINGS\n")
	printFlag("ACK", f.IsAck(), in)
	f.ForeachSetting(func(s http2.Setting) error {
		printKeyValue(s.ID.String(), s.Val, in)
		return nil
	})
}

func printUpdateFrame(f *http2.WindowUpdateFrame, in bool) {
	color := dirColor(in)
	defer color.Printf("|\n")
	color.Printf("+- WINDOW_UPDATE")
	fmt.Printf(" #%v", f.StreamID)
	fmt.Println()
}

func printSettingACK() {
	color := dirColor(false)
	defer color.Printf("|\n")
	color.Printf("+- SETTINGS\n")
	color.Printf("| 🏁 ACK:")
	fmt.Printf("%v\n", true)
}

func printHeadersFrame(f *http2.HeadersFrame, in bool) {
	color := dirColor(in)
	defer color.Printf("|\n")
	color.Printf("+- HEADERS")
	fmt.Printf(" #%v", f.StreamID)
	fmt.Println()

	printFlag("END_STREAM", f.Flags.Has(http2.FlagHeadersEndStream), in)
	printFlag("END_HEADERS", f.Flags.Has(http2.FlagHeadersEndHeaders), in)
	printFlag("PADDED", f.Flags.Has(http2.FlagHeadersPadded), in)
	printFlag("PRIORITY", f.Flags.Has(http2.FlagHeadersPriority), in)

	hf, _ := decoder.DecodeFull(f.HeaderBlockFragment())
	for _, h := range hf {
		printKeyValue(h.Name, h.Value, in)
	}
}

func printDataFrame(f *http2.DataFrame, in bool) {
	color := dirColor(in)
	defer color.Printf("|\n")
	color.Printf("+- DATA")
	fmt.Printf(" #%v", f.StreamID)
	fmt.Println()

	printFlag("END_STREAM", f.Flags.Has(http2.FlagDataEndStream), in)
	printFlag("PADDED", f.Flags.Has(http2.FlagDataPadded), in)
	printKeyValue("Data", fmt.Sprintf("%q", f.Data()), in)
}

func printKeyValue(key string, value interface{}, in bool) {
	color := dirColor(in)
	color.Printf("| %s:", key)
	valueStr, ok := value.(string)
	if ok && len(key)+len(valueStr) > 50 {
		value = valueStr[0:(50-len(key))] + "..."
	}
	fmt.Printf(" %v", value)
	fmt.Println()
}

func printFlag(key string, value interface{}, in bool) {
	color := dirColor(in)
	color.Printf("| 🏁 %s:", key)
	fmt.Printf(" %v", value)
	fmt.Println()
}

// Les requêtes HTTP2 fonctionnent avec un système de frames
func handleHTTP22(conn net.Conn) {
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
				respondHTTP2(r, streamID, framer)
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

// Lit un certain nombre d'octet dans un reader
func readBytes(r io.Reader, n int) ([]byte, error) {
	buffer := make([]byte, n)
	b, err := io.ReadFull(r, buffer)

	if err == io.ErrUnexpectedEOF {
		return buffer[:b], fmt.Errorf("Unexpected end of buffer")
	}

	if err != nil && !strings.Contains(err.Error(), "unknown certificate") {
		return buffer, err
	}

	return buffer, nil
}

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

func respondHTTP2(r *Request, streamID uint32, framer *http2.Framer) {
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
	content, err := os.ReadFile("public/" + r.Path)
	if err != nil {
		framer.WriteData(streamID, true, []byte{})
		log.Printf("Cannot read file %s", r.Path)
		return
	}

	framer.WriteData(streamID, true, content)
}
