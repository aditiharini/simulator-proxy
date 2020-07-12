package main

import (
	"crypto/rand"
	"flag"
	"net"
)

func main() {
	dest := flag.String("dest", "", "destination ip address and port")
	numPackets := flag.Int("count", 1, "number of packets to send")
	packetSize := flag.Int("size", 10, "size of each packet sent")
	flag.Parse()

	addr, err := net.ResolveUDPAddr("udp", *dest)
	if err != nil {
		panic(err)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		panic(err)
	}

	packet := make([]byte, *packetSize)
	for i := 0; i < *numPackets; i++ {
		bytesGenerated, err := rand.Read(packet)
		if err != nil {
			panic(err)
		}
		if bytesGenerated != *packetSize {
			panic("incorrect number of random bytes written")
		}
		bytesWritten, err := conn.Write(packet)
		if err != nil {
			panic(err)
		}
		if bytesWritten != *packetSize {
			panic("incorrect number of bytes written to device")
		}
	}

}
