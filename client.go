package main

import (
	"fmt"
	"log"
	"net"

	"go.bug.st/serial"
)

func runClient(port serial.Port, portMappings []PortMapping) {
	log.Println("Running as client")
	sendHello(port)
	waitHello(port)
	log.Println("Handshake complete")

	go serialWriter(port)

	for _, mapping := range portMappings {
		go listenAndProxy(mapping)
	}

	for {
		packet, err := readPacket(port)
		if err != nil {
			log.Printf("Error reading packet: %v", err)
			continue
		}
		handleSerialPacket(packet)
	}
}

func listenAndProxy(mapping PortMapping) {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", mapping.LocalHost, mapping.LocalPort))
	if err != nil {
		log.Printf("Failed to listen on port %s:%d: %v", mapping.LocalHost, mapping.LocalPort, err)
		return
	}
	defer listener.Close()

	log.Printf("Listening on port %s:%d, proxying to remote port %s:%d", mapping.LocalHost, mapping.LocalPort, mapping.RemoteHost, mapping.RemotePort)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}
		cid := getNextConnectionID()

		connectionsMutex.Lock()
		connections[cid] = conn
		connectionsMutex.Unlock()

		if err := initProxy(mapping, cid); err != nil {
			log.Printf("Failed to init proxy for %s:%d->%s:%d: %v", mapping.LocalHost, mapping.LocalPort, mapping.RemoteHost, mapping.RemotePort, err)
			connectionsMutex.Lock()
			delete(connections, cid)
			connectionsMutex.Unlock()
			continue
		}
		go handleConnection(conn, cid)
	}
}

func initProxy(mapping PortMapping, connectionID uint32) error {
	payload := fmt.Sprintf("%s:%d:%s:%d:%d", mapping.LocalHost, mapping.LocalPort, mapping.RemoteHost, mapping.RemotePort, connectionID)
	response, err := sendRequest(PacketTypeInitProxy, []byte(payload))
	if err != nil {
		return err
	}
	if !response.Success {
		return fmt.Errorf("failed to init proxy: %s", response.Message)
	}
	return nil
}
