package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.bug.st/serial"
)

type PortMapping struct {
	LocalHost  string
	LocalPort  int
	RemoteHost string
	RemotePort int
}

type ResponsePacket struct {
	RequestID uint32
	Success   bool
	Message   string
}

var (
	nextRequestID        uint32 = 1
	requestIDMutex       sync.Mutex
	pendingResponses     = make(map[uint32]chan ResponsePacket)
	pendingResponseMutex sync.Mutex
	outgoingPackets      = make(chan OutgoingPacket, 100)
)

func getNextRequestID() uint32 {
	requestIDMutex.Lock()
	defer requestIDMutex.Unlock()
	id := nextRequestID
	nextRequestID++
	return id
}

func sendRequest(packetType byte, payload []byte) (ResponsePacket, error) {
	requestID := getNextRequestID()
	responseChan := make(chan ResponsePacket, 1)

	pendingResponseMutex.Lock()
	pendingResponses[requestID] = responseChan
	pendingResponseMutex.Unlock()

	outgoingPackets <- OutgoingPacket{
		Type: packetType,
		Payload: append([]byte{
			byte(requestID >> 24),
			byte(requestID >> 16),
			byte(requestID >> 8),
			byte(requestID),
		}, payload...),
	}

	select {
	case response := <-responseChan:
		return response, nil
	case <-time.After(10 * time.Second):
		pendingResponseMutex.Lock()
		delete(pendingResponses, requestID)
		pendingResponseMutex.Unlock()
		return ResponsePacket{}, fmt.Errorf("request timed out")
	}
}

func sendResponse(requestID uint32, success bool, message string) {
	response := ResponsePacket{
		RequestID: requestID,
		Success:   success,
		Message:   message,
	}
	payload, _ := json.Marshal(response)
	outgoingPackets <- OutgoingPacket{
		Type:    PacketTypeResponse,
		Payload: payload,
	}
}

func getPacketRequestID(packet Packet) (uint32, []byte) {
	return binary.BigEndian.Uint32(packet.Payload), packet.Payload[4:]
}

func splitHostPort(addr string) (string, int, error) {
	parts := strings.Split(addr, ":")

	// if parts is one, then assume host is localhost
	if len(parts) == 1 {
		port, err := strconv.Atoi(parts[0])
		if err != nil {
			return "", 0, fmt.Errorf("invalid port: %s", parts[0])
		}
		return "localhost", port, nil
	}

	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid address: %s", addr)
	}

	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, fmt.Errorf("invalid port: %s", parts[1])
	}
	return parts[0], port, nil
}

func parsePortMappings(mappings string) ([]PortMapping, error) {
	var result []PortMapping
	pairs := strings.Split(mappings, ",")
	for _, pair := range pairs {
		ports := strings.Split(pair, "-")
		if len(ports) != 2 {
			return nil, fmt.Errorf("invalid port mapping: %s", pair)
		}

		localHost, localPort, err := splitHostPort(ports[0])
		if err != nil {
			return nil, fmt.Errorf("invalid local port: %s", ports[0])
		}

		remoteHost, remotePort, err := splitHostPort(ports[1])
		if err != nil {
			return nil, fmt.Errorf("invalid remote port: %s", ports[1])
		}

		result = append(result, PortMapping{
			LocalHost:  localHost,
			LocalPort:  localPort,
			RemoteHost: remoteHost,
			RemotePort: remotePort,
		})
	}
	return result, nil
}

func handleResponse(packet Packet, verbose bool) {
	var response ResponsePacket
	if err := json.Unmarshal(packet.Payload, &response); err != nil {
		log.Printf("Error unmarshaling response packet: %v", err)
		return
	}

	if verbose {
		log.Printf("Received response packet: %d", response.RequestID)
	}

	pendingResponseMutex.Lock()
	if ch, ok := pendingResponses[response.RequestID]; ok {
		ch <- response
		delete(pendingResponses, response.RequestID)
	}
	pendingResponseMutex.Unlock()
}

func sendHello(w serial.Port) {
	sendPacket(w, Packet{Type: PacketTypeHello, Payload: []byte("HELLO")})
}

func waitHello(r io.Reader) error {
	packet, err := readPacket(r, false)
	if err != nil {
		return err
	}
	if packet.Type != PacketTypeHello {
		return fmt.Errorf("expected hello packet, got type %d", packet.Type)
	}
	log.Printf("Received hello: %s", string(packet.Payload))
	return nil
}

func handleSerialPacket(packet Packet, verbose bool) {
	switch packet.Type {
	case PacketTypeHello:
		log.Printf("Received hello: %s", string(packet.Payload))
	case PacketTypeInitProxy:
		_, addr := getPacketRequestID(packet)
		log.Printf("Initiating proxy to %s", addr)
		handleInitProxy(packet)
	case PacketTypeData:
		if len(packet.Payload) < 4 {
			log.Printf("Invalid data packet: payload too short")
			return
		}
		connectionID := binary.BigEndian.Uint32(packet.Payload[:4])
		data := packet.Payload[4:]

		connectionsMutex.Lock()
		conn, ok := connections[connectionID]
		connectionsMutex.Unlock()

		if !ok {
			log.Printf("No connection found for connection ID %d", connectionID)
			return
		}

		if verbose {
			log.Printf("Sending %d bytes to connection %d", len(data), connectionID)
		}
		_, err := conn.Write(data)
		if err != nil {
			log.Printf("Error writing to TCP connection: %v", err)
		}
	case PacketTypeClosePort:
		connectionID, err := strconv.ParseUint(string(packet.Payload), 10, 32)
		if err != nil {
			log.Printf("Invalid close port packet: %v", err)
			return
		}

		connectionsMutex.Lock()
		conn, ok := connections[uint32(connectionID)]
		if ok {
			delete(connections, uint32(connectionID))
			conn.Close()
		} else {
			log.Printf("PacketTypeClosePort: No connection found for connection ID %d", connectionID)
		}
		connectionsMutex.Unlock()
	case PacketTypeResponse:
		handleResponse(packet, verbose)
	default:
		log.Printf("Unknown packet type: %d", packet.Type)
	}
}
