package main

import (
	"crypto/rand"
	"flag"
	"net"
	"time"
)

func main() {
	// TODO(aditi) It would be nice to have subcommands with some shared flags.
	// Not sure how to do this.
	dest := flag.String("dest", "", "destination ip address and port")
	numPackets := flag.Int("count", 1, "number of packets to send")
	packetSize := flag.Int("size", 10, "size of each packet sent")
	waitTime := flag.Int("wait", 0, "ms to wait between sending packets")
	isBursty := flag.Bool("bursty", false, "send bursty traffic")
	packetsPerBurst := flag.Int("packetsPerBurst", 100, "number of packets in each burst")

	flag.Parse()

	addr, err := net.ResolveUDPAddr("udp", *dest)
	if err != nil {
		panic(err)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		panic(err)
	}

	packetsLeftInBurst := *packetsPerBurst
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

		if *isBursty {
			if packetsLeftInBurst == 0 {
				time.Sleep(time.Millisecond * time.Duration((*waitTime)*(*packetsPerBurst)))
				packetsLeftInBurst = *packetsPerBurst
			} else {
				packetsLeftInBurst--
			}

		} else if *waitTime > 0 {
			time.Sleep(time.Millisecond * time.Duration(*waitTime))
		}
	}

}
