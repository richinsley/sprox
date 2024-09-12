package main

import (
	"encoding/binary"
	"io"
	"log"

	"go.bug.st/serial"
)

const (
	PacketTypeHello     = 0
	PacketTypeInitProxy = 1
	PacketTypeData      = 2
	PacketTypeClosePort = 3
	PacketTypeResponse  = 4
	MaxPayloadSize      = 1024 * 8
	HeaderSize          = 3 // 1 byte for type, 2 bytes for payload size
)

type Packet struct {
	Type      byte
	RequestID uint32
	Payload   []byte
}

type OutgoingPacket struct {
	Type    byte
	Payload []byte
}

func sendPacket(w serial.Port, packet Packet) error {
	header := make([]byte, HeaderSize)
	header[0] = packet.Type
	binary.BigEndian.PutUint16(header[1:], uint16(len(packet.Payload)))

	// Write header
	written := 0
	for written < len(header) {
		n, err := w.Write(header[written:])
		if err != nil {
			return err
		}
		written += n
	}

	// Write payload
	written = 0
	for written < len(packet.Payload) {
		n, err := w.Write(packet.Payload[written:])
		if err != nil {
			return err
		}
		written += n
	}

	// Wait until all data in the buffer are sent
	err := w.Drain()
	if err != nil {
		return err
	}

	return nil
}

// func sendPacket(w serial.Port, packet Packet) error {
// 	header := make([]byte, HeaderSize)
// 	header[0] = packet.Type
// 	binary.BigEndian.PutUint16(header[1:], uint16(len(packet.Payload)))

// 	n, err := w.Write(header)
// 	if err != nil {
// 		return err
// 	}
// 	if n != len(header) {
// 		return io.ErrShortWrite
// 	}

// 	n, err = w.Write(packet.Payload)
// 	if err != nil {
// 		return err
// 	}
// 	if n != len(packet.Payload) {
// 		return io.ErrShortWrite
// 	}
// 	w.Drain()

// 	return err
// }

func readPacket(r io.Reader, verbose bool) (Packet, error) {
	header := make([]byte, HeaderSize)
	if _, err := io.ReadFull(r, header); err != nil {
		return Packet{}, err
	}

	packetType := header[0]
	payloadSize := binary.BigEndian.Uint16(header[1:])

	payload := make([]byte, payloadSize)
	n, err := io.ReadFull(r, payload)
	if err != nil {
		return Packet{}, err
	}

	if n != int(payloadSize) {
		return Packet{}, io.ErrUnexpectedEOF
	}

	retv := Packet{Type: packetType, Payload: payload}
	if verbose {
		log.Printf("Read serial packet type: %d payload size: %d", packetType, n)
	}
	return retv, nil
}
