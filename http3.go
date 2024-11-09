package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/k0kubun/pp/v3"
	"github.com/quic-go/qpack"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/quicvarint"
)

const HTTP3 = "h3"

const (
	streamTypeControlStream      = 0
	streamTypePushStream         = 1
	streamTypeQPACKEncoderStream = 2
	streamTypeQPACKDecoderStream = 3
)

const (
	dataFrameType   = 0x00
	headerFrameType = 0x01
)

type ErrCode quic.ApplicationErrorCode

const (
	ErrCodeNoError              ErrCode = 0x100
	ErrCodeGeneralProtocolError ErrCode = 0x101
	ErrCodeInternalError        ErrCode = 0x102
	ErrCodeStreamCreationError  ErrCode = 0x103
	ErrCodeClosedCriticalStream ErrCode = 0x104
	ErrCodeFrameUnexpected      ErrCode = 0x105
	ErrCodeFrameError           ErrCode = 0x106
	ErrCodeExcessiveLoad        ErrCode = 0x107
	ErrCodeIDError              ErrCode = 0x108
	ErrCodeSettingsError        ErrCode = 0x109
	ErrCodeMissingSettings      ErrCode = 0x10a
	ErrCodeRequestRejected      ErrCode = 0x10b
	ErrCodeRequestCanceled      ErrCode = 0x10c
	ErrCodeRequestIncomplete    ErrCode = 0x10d
	ErrCodeMessageError         ErrCode = 0x10e
	ErrCodeConnectError         ErrCode = 0x10f
	ErrCodeVersionFallback      ErrCode = 0x110
	ErrCodeDatagramError        ErrCode = 0x33
)

// Crée un serveur HTTP2 qui informe du support de l'HTTP3
func broadcastHTTP3(config *tls.Config) {
	h1and2 := http.Server{
		Addr:      "localhost:8000",
		TLSConfig: config,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			pp.Println("Handle HTTP2")
			w.Header().Set("alt-svc", "h3=\":8000\"; ma=86400, quic=\":8000\"; ma=86400")
			return
		})}
	err := h1and2.ListenAndServeTLS("cert.pem", "key.pem")
	if err != nil {
		log.Printf("Error listen and serving h1&2 %v", err)
		return
	}
}

// Gère une requête HTTP3
// Détail de l'échange QUIC : https://quic.xargs.org/
// Detail du protocol : https://http3-explained.haxx.se/en
// Pour tester : curl --http3 -v --insecure https://localhost
func handleHTTP3(conn quic.Connection) error {
	fmt.Printf("⦿\n")
	defer fmt.Printf("x\n")
	// On envoit la frame de "SETTINGS"
	str, err := conn.OpenUniStreamSync(context.Background())
	if err != nil {
		return fmt.Errorf("impossible d'ouvrir un stream unifirectionnel %w", err)
	}
	b := make([]byte, 0, 64)
	b = quicvarint.Append(b, streamTypeControlStream)
	b = quicvarint.Append(b, 0x4)
	b = quicvarint.Append(b, 0)
	str.Write(b)

	// On gère le flux unidirectionnel pour recevoir les settings
	go listenUniStream(conn)

	// Flux principal (bidirectionnel) qui contiendra requête et réponse
	var wg sync.WaitGroup
	for {
		str, err := conn.AcceptStream(context.Background())
		if err != nil {
			log.Printf("cannot accept bi stream %v", err)
			break
		}
		go handleRequest(conn, str)
	}
	wg.Wait()
	return nil
}

func listenUniStream(conn quic.Connection) {
	for {
		str, err := conn.AcceptUniStream(context.Background())
		if err != nil {
			log.Printf("cannot accept uni stream %v", err.Error())
			return
		}

		go func(str quic.ReceiveStream) {
			streamType, err := quicvarint.Read(quicvarint.NewReader(str))
			if err != nil {
				log.Printf("cannot read stream type %v", err.Error())
				return
			}
			fp := NewFrameParser(str, nil)
			switch streamType {
			case streamTypeControlStream:
				f, err := fp.NextFrame()
				if err != nil {
					log.Printf("cannot parse next frame %v\n", err.Error())
				}
				switch f.(type) {
				case SettingsFrame:
					printH3Frame(f, true)
				default:
					log.Printf("control stream expected, got %+v\n", f)
				}
			// On ne gère que le flux de contrôle
			default:
				return
			}
		}(str)
	}
}

func handleRequest(conn quic.Connection, str quic.Stream) {
	fmt.Printf("+==============\n")
	fmt.Printf("+= Stream #%v =\n", str.StreamID())
	fmt.Printf("+==============\n")
	defer fmt.Printf("+==============\n")
	decoder := qpack.NewDecoder(func(hf qpack.HeaderField) {})
	fp := NewFrameParser(str, decoder)
	f, err := fp.NextFrame()
	if err != nil {
		log.Printf("Cannot read header frame %w", err)
		return
	}
	hf, ok := f.(HeadersFrame)
	if !ok {
		conn.CloseWithError(quic.ApplicationErrorCode(ErrCodeFrameUnexpected), "expected first frame to be a HEADERS frame")
		return
	}
	printH3Frame(hf, true)

	if hf.Header(":method", "GET") != "GET" {
		f, err = fp.NextFrame()
		if err != nil {
			log.Printf("Cannot read data frame %w", err)
			return
		}
		printH3Frame(f, true)
	}
	hf, df := framesFromRequest(hf)
	printH3Frame(hf, false)
	hf.Write(str)
	printH3Frame(df, false)
	df.Write(str)
	str.Close()
}

// Génère les frames à renvoyer en fonction de la requêt
func framesFromRequest(f HeadersFrame) (HeadersFrame, DataFrame) {
	path := f.Header(":path", "/")
	if strings.HasSuffix(path, "/") {
		path = path + "index.html"
	}
	path = strings.Trim(path, "/")
	content, _ := os.ReadFile("public/" + path)
	return HeadersFrame{
			Headers: []qpack.HeaderField{
				{Name: ":status", Value: "200"},
				{Name: "content-type", Value: "text/" + getFileExtension(path)},
			},
		}, DataFrame{
			Data: content,
		}
}

func printH3Frame(f interface{}, in bool) {
	printMu.Lock()
	defer printMu.Unlock()
	switch f := f.(type) {
	case SettingsFrame:
		printH3SettingFrame(in)
	case HeadersFrame:
		printH3HeadersFrame(f, in)
	case DataFrame:
		printH3DataFrame(f, in)
	default:
		fmt.Println("Cannot print unknown frame %t", f)
	}
}

func printH3SettingFrame(in bool) {
	color := dirColor(in)
	defer color.Printf("|\n")
	color.Printf("+- SETTINGS\n")
}

func printH3HeadersFrame(f HeadersFrame, in bool) {
	color := dirColor(in)
	defer color.Printf("|\n")
	color.Printf("+- HEADERS")
	fmt.Println()

	for _, h := range f.Headers {
		printKeyValue(h.Name, h.Value, in)
	}
}

func printH3DataFrame(f DataFrame, in bool) {
	color := dirColor(in)
	defer color.Printf("|\n")
	color.Printf("+- Data")
	fmt.Println()
	color.Printf("| ...\n")
}

type framerParser struct {
	str     io.Reader
	decoder *qpack.Decoder
}

func NewFrameParser(str io.Reader, decoder *qpack.Decoder) *framerParser {
	return &framerParser{
		str:     str,
		decoder: decoder,
	}
}

func (fp *framerParser) NextFrame() (HTTP3Frame, error) {
	qr := quicvarint.NewReader(fp.str)
	for {
		t, err := quicvarint.Read(qr)
		if err == io.EOF {
			return nil, err
		}
		if err != nil {
			return nil, fmt.Errorf("impossible de lire le type de la frame, %s", err.Error())
		}

		l, err := quicvarint.Read(qr)
		if err == io.EOF {
			return nil, err
		}
		if err != nil {
			return nil, fmt.Errorf("impossible de lire la longueur de la frame, %s", err.Error())
		}

		switch t {
		case dataFrameType:
			buf := make([]byte, l)
			qr.Read(buf)
			return DataFrame{
				Data: buf,
			}, nil
		case headerFrameType:
			buf := make([]byte, l)
			qr.Read(buf)
			fields, err := fp.decoder.DecodeFull(buf)
			if err != nil {
				return nil, fmt.Errorf("impossible de décoder les en tête, %s", err.Error())
			}
			return HeadersFrame{
				Headers: fields,
			}, nil
		case 0x04:
			buf := make([]byte, l)
			qr.Read(buf)
			return NewSettingsFrame(buf), nil
		}
		if _, err := io.CopyN(io.Discard, qr, int64(l)); err != nil {
			return nil, err
		}
	}
}

type HTTP3Frame interface {
}

type HeadersFrame struct {
	Headers []qpack.HeaderField
}

type DataFrame struct {
	Data []byte
}

type SettingsFrame struct {
	Settings []Setting
}

type Setting struct {
	Identifier uint64
	Value      uint64
}

func NewSettingsFrame(buf []byte) SettingsFrame {
	r := bytes.NewReader(buf)
	f := SettingsFrame{}
	for {
		id, err := quicvarint.Read(r)
		if err != io.EOF {
			break
		}
		v, err := quicvarint.Read(r)
		if err != io.EOF {
			break
		}
		f.Settings = append(f.Settings, Setting{id, v})
	}
	return f
}

func (f HeadersFrame) Write(w io.Writer) {
	var headers bytes.Buffer
	enc := qpack.NewEncoder(&headers)

	for _, field := range f.Headers {
		err := enc.WriteField(field)
		if err != nil {
			log.Printf("impossible d'écrire l'en tête dans l'encodeur %v", err)
		}
	}

	buf := make([]byte, 0, 16+headers.Len())
	buf = quicvarint.Append(buf, headerFrameType)
	buf = quicvarint.Append(buf, uint64(headers.Len()))
	buf = append(buf, headers.Bytes()...)
	n, err := w.Write(buf)
	if err != nil {
		log.Printf("impossible d'écrire l'en tête %v %v\n", n, err)
	}
}

func (f HeadersFrame) Header(name string, base string) string {
	for _, field := range f.Headers {
		if field.Name == name {
			return field.Value
		}
	}
	return base
}

func (f DataFrame) Write(w io.Writer) {
	out := make([]byte, 0)
	out = quicvarint.Append(out, dataFrameType)
	out = quicvarint.Append(out, uint64(len(f.Data)))
	w.Write(out)
	w.Write(f.Data)
}
