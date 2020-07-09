package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	// Address to listen on
	addr, err := net.ResolveUDPAddr("udp", os.Args[1])
	if err != nil {
		panic(err)
	}

	conn, err := net.ListenUDP("udp", addr)

	if err != nil {
		panic(err)
	}

	defer conn.Close()

	buffer := make([]byte, 1024)
	for {
		numBytes, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			panic(err)
		}

		fmt.Println(string(buffer[:numBytes]))
	}
}
