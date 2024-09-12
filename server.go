package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"go.bug.st/serial"
)

func runServer(port serial.Port, portMappings []PortMapping, verbose bool) {
	log.Println("Running as server")
	waitHello(port)
	sendHello(port)
	log.Println("Handshake complete")

	go serialWriter(port)

	for _, mapping := range portMappings {
		go listenAndProxy(mapping, verbose)
	}

	for {
		packet, err := readPacket(port, verbose)
		if err != nil {
			log.Printf("Error reading packet: %v", err)
			os.Exit(1)
		}
		handleSerialPacket(packet, verbose)
	}
}

func handleInitProxy(packet Packet) {
	requestID, payload := getPacketRequestID(packet)
	parts := strings.Split(string(payload), ":")
	if len(parts) != 5 {
		sendResponse(requestID, false, "invalid payload format")
		return
	}
	remoteHost := parts[0]
	remotePort, _ := strconv.Atoi(parts[1])

	localHost := parts[2]
	localPort, _ := strconv.Atoi(parts[3])

	connectionID, _ := strconv.ParseUint(parts[4], 10, 32)

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", localHost, localPort))
	if err != nil {
		sendResponse(requestID, false, fmt.Sprintf("failed to dial port %d: %v", localPort, err))
		return
	}

	sendResponse(requestID, true, "proxy initialized successfully")

	go handleServerConnection(conn, remoteHost, remotePort, uint32(connectionID))
}

func serialWriter(serialPort serial.Port) {
	for packet := range outgoingPackets {
		err := sendPacket(serialPort, Packet{Type: packet.Type, Payload: packet.Payload})
		if err != nil {
			log.Printf("Error sending packet: %v", err)
		}
		serialPort.Drain()
	}
}
