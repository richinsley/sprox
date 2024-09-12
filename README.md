# USB Serial Port Proxy (sprox)
A simple tcp/ip port proxy via serial connection

## Overview

This project, USB Serial Port Proxy (sprox), is designed to utilize a Raspberry Pi set up in USB serial gadget mode to create a proxy for TCP ports. It allows you to forward ports from a target machine (server) through a USB serial connection to a Raspberry Pi (client), effectively exposing these ports on the Raspberry Pi.

The primary use case for this project is to work on embedded Linux systems, allowing modification of the TCP/IP stack and devices while still being able to use tools like gdbserver for debugging code on the embedded system.


## Features

- Port forwarding through USB serial connection
- Bi-directional communication between server and client
- Flexible port mapping configuration
- Support for multiple simultaneous port forwards

## Prerequisites

- A Raspberry Pi configured in USB serial gadget mode
- A target machine (embedded Linux system) with a USB port
- Go programming environment (for building the project)

## Usage

### Server Side (Target Machine)

Run the server on your target machine (ie embedded Linux system):

```bash
./sprox -server -port /dev/ttyGS0 -baud 115200 -ports 9922-192.168.0.31:22,9980-127.0.0.1:8080
```

### Client Side (Raspberry Pi)

Run the client on your Raspberry Pi:

```bash
./sprox -port /dev/ttyGS0 -baud 1152000 -ports 9999-8080,2537-1537
```

This example maps:
- Local port 9999 to remote port 8080
- Local port 2537 to remote port 1537

## Port Mapping Syntax

The port mapping syntax is as follows:
```
[local_port]->[remote_host]:[remote_port]
```

If `<remote_host>` is omitted, it defaults to localhost.  Use 0.0.0.0 to expose port on all devices.

Multiple mappings can be specified by separating them with commas.

## Building

To build the project, ensure you have Go installed, then run:

```bash
go build -o sprox
```

## Use Case Example

1. Set up your Raspberry Pi in USB serial gadget mode and connect it to your embedded Linux system via USB.
2. Run the server on your embedded Linux system, forwarding necessary ports (e.g., gdbserver port).
3. Run the client on your Raspberry Pi.
4. You can now connect to the forwarded ports on your Raspberry Pi, which will proxy the connections to your embedded Linux system.
5. This setup allows you to modify the TCP/IP stack or devices on your embedded system while still maintaining the ability to use debugging tools through the USB serial connection.

## Notes

- For USB gadget mode, the baud rate is ignored and connection runs at the speed of USB connection.
- The USB serial device name may vary depending on your system configuration.
