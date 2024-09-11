package main

import (
	"encoding/binary"
	"io"
)

const (
	PacketTypeHello     = 0
	PacketTypeInitProxy = 1
	PacketTypeData      = 2
	PacketTypeClosePort = 3
	PacketTypeResponse  = 4
	MaxPayloadSize      = 1024 * 16
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

func sendPacket(w io.Writer, packet Packet) error {
	header := make([]byte, HeaderSize)
	header[0] = packet.Type
	binary.BigEndian.PutUint16(header[1:], uint16(len(packet.Payload)))

	if _, err := w.Write(header); err != nil {
		return err
	}
	_, err := w.Write(packet.Payload)
	return err
}

func readPacket(r io.Reader) (Packet, error) {
	header := make([]byte, HeaderSize)
	if _, err := io.ReadFull(r, header); err != nil {
		return Packet{}, err
	}

	packetType := header[0]
	payloadSize := binary.BigEndian.Uint16(header[1:])

	payload := make([]byte, payloadSize)
	if _, err := io.ReadFull(r, payload); err != nil {
		return Packet{}, err
	}

	return Packet{Type: packetType, Payload: payload}, nil
}
