package main

import (
	"flag"
	"log"

	"go.bug.st/serial"
)

func main() {
	portName := flag.String("port", "", "Serial port name")
	baudRate := flag.Int("baud", 115200, "Baud rate")
	isServer := flag.Bool("server", false, "Run as server")
	portMappings := flag.String("ports", "", "Comma-separated list of local:remote port mappings (e.g., 80:8081,1537:2537)")
	flag.Parse()

	if *portName == "" {
		log.Fatal("Please specify a serial port using the -port flag")
	}

	log.Printf("Connected to %s at %d baud", *portName, *baudRate)

	var err error
	if *isServer {
		var mappings []PortMapping
		if *portMappings != "" {
			mappings, err = parsePortMappings(*portMappings)
			if err != nil {
				log.Fatalf("Failed to parse port mappings: %v", err)
			}
		}

		port, err := serial.Open(*portName, &serial.Mode{BaudRate: *baudRate})
		if err != nil {
			log.Fatalf("Failed to open serial port: %v", err)
		}
		defer port.Close()

		runServer(port, mappings)
	} else {
		var mappings []PortMapping
		if *portMappings != "" {
			mappings, err = parsePortMappings(*portMappings)
			if err != nil {
				log.Fatalf("Failed to parse port mappings: %v", err)
			}
		}

		port, err := serial.Open(*portName, &serial.Mode{BaudRate: *baudRate})
		if err != nil {
			log.Fatalf("Failed to open serial port: %v", err)
		}
		defer port.Close()
		runClient(port, mappings)
	}
}
