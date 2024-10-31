package main

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/k0kubun/pp/v3"
)

const frameTypeSetting = 0x04

// https://httpwg.org/specs/rfc7540.html#ConnectionHeader

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

func readByte(r io.Reader) (byte, error) {
	bytes, err := readBytes(r, 1)
	if err != nil {
		return 0, err
	}

	return bytes[0], nil
}

type Frame struct {
	Length     int32
	Type       []byte
	Flags      []byte
	R          byte
	Identifier []byte
	Payload    []byte
}

func NewFrame(r io.Reader) interface{} {
	length, err := readUint24(r)
	if err != nil {
		log.Println("Cannot read length for the frame")
	}
	frameType, err := readByte(r)
	if err != nil {
		log.Println("Cannot read length for the frame")
	}

	flags, err := readByte(r)
	if err != nil {
		log.Println("Cannot read flag for the frame")
	}

	_, err = readBytes(r, 4)
	if err != nil {
		log.Println("Cannot read streamidentifier")
	}

	pp.Printf("> Frame type: %v (%v)\n", frameType, length)
	if frameType == 0x04 {
		return NewSettingFrame(r, length)
	}
	if frameType == 0x01 {
		//	return NewHeadersFrame(r, length, flags)
	}

	UNUSED(flags)

	// Unknown frame type, skip the value
	readBytes(r, int(length))

	return nil
}

func NewSettingFrame(r io.Reader, length uint32) interface{} {
	paramsLength := int(length / 6)
	for i := 0; i < paramsLength; i++ {
		idenfitier, err := readBytes(r, 2)
		value, err := readBytes(r, 4)
		UNUSED(idenfitier, value, err)
	}
	return nil
}

func NewHeadersFrame(r io.Reader, length uint32, flags byte) interface{} {
	isPadded := readBit(flags, 4)
	isPriority := readBit(flags, 2)

	if isPadded {
		padLength, err := readByte(r)
		UNUSED(padLength, err)
	}

	if isPriority {
		deps, err := readBytes(r, 4)
		weight, err := readBytes(r, 1)
		UNUSED(deps, err, weight)
	}

	return nil
}

func readUint24(r io.Reader) (uint32, error) {
	buf := make([]byte, 3)
	_, err := io.ReadFull(r, buf)
	if err != nil {
		return 0, err
	}

	return uint32(buf[0])<<16 | uint32(buf[1])<<8 | uint32(buf[2]), nil
}

// Lit un bit dans un octet (true si bit = 1)
func readBit(b byte, pos uint) bool {
	if pos > 7 {
		panic("Bit position must be between 0 and 7")
	}
	return (b & (1 << (7 - pos))) != 0
}
