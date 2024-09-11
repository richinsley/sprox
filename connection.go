package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

var (
	nextConnectionID  uint32 = 1
	connectionIDMutex sync.Mutex

	connections      = make(map[uint32]net.Conn)
	connectionsMutex sync.Mutex
)

func getNextConnectionID() uint32 {
	connectionIDMutex.Lock()
	defer connectionIDMutex.Unlock()
	id := nextConnectionID
	nextConnectionID++
	return id
}

func handleConnection(conn net.Conn, connectionID uint32) {
	defer conn.Close()
	buf := make([]byte, MaxPayloadSize)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading from TCP connection: %v", err)
			}
			break
		}
		outgoingPackets <- OutgoingPacket{
			Type:    PacketTypeData,
			Payload: append([]byte{byte(connectionID >> 24), byte(connectionID >> 16), byte(connectionID >> 8), byte(connectionID)}, buf[:n]...),
		}
	}
	outgoingPackets <- OutgoingPacket{
		Type:    PacketTypeClosePort,
		Payload: []byte(fmt.Sprintf("%d", connectionID)),
	}
}

func handleServerConnection(conn net.Conn, localHost string, localPort int, connectionID uint32) {
	defer conn.Close()

	remoteAddr := conn.RemoteAddr().String()
	log.Printf("New connection from %s to port %s:%d", remoteAddr, localHost, localPort)

	connectionsMutex.Lock()
	connections[connectionID] = conn
	connectionsMutex.Unlock()

	defer func() {
		connectionsMutex.Lock()
		delete(connections, connectionID)
		connectionsMutex.Unlock()
	}()

	buf := make([]byte, MaxPayloadSize)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading from TCP connection %s: %v", remoteAddr, err)
			}
			break
		}

		payload := append([]byte{byte(connectionID >> 24), byte(connectionID >> 16), byte(connectionID >> 8), byte(connectionID)}, buf[:n]...)
		outgoingPackets <- OutgoingPacket{
			Type:    PacketTypeData,
			Payload: payload,
		}
	}

	outgoingPackets <- OutgoingPacket{
		Type:    PacketTypeClosePort,
		Payload: []byte(fmt.Sprintf("%d", localPort)),
	}
}
