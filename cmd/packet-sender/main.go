package main

import (
	"net"
	"os"
)

func main() {

	addr, err := net.ResolveUDPAddr("udp", os.Args[1])
	if err != nil {
		panic(err)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		panic(err)
	}

	for i := 0; i < 1; i++ {
		msg := "hi"
		_, err := conn.Write([]byte(msg))
		if err != nil {
			panic(err)
		}
	}

}
