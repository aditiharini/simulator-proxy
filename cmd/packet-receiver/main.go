package main

import (
	"flag"
	"fmt"
	"net"
)

func main() {
	listenAddr := flag.String("listen-on", "", "address and port to listen on ")
	flag.Parse()

	addr, err := net.ResolveUDPAddr("udp", *listenAddr)
	if err != nil {
		panic(err)
	}

	conn, err := net.ListenUDP("udp", addr)

	if err != nil {
		panic(err)
	}

	defer conn.Close()

	buffer := make([]byte, 2048)
	packetNum := 0
	for {
		numBytes, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			panic(err)
		}

		packetNum++
		fmt.Println(packetNum, ":", buffer[:numBytes])
	}
}
