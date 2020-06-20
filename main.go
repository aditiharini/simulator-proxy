package main

import (
	"fmt"
	"net"
	"os/exec"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/songgao/water"
)

type Address = int

type Packet struct {
	src         Address
	dst         Address
	hopsLeft    int
	data        []byte
	arrivalTime time.Time
}

type Simulator interface {
	processOutgoingPackets()
	processIncomingPackets()
	writeToDestination(Packet)
}

type BroadcastSimulator struct {
	queues   map[Address]([]LinkEmulator) // Map src to list of links
	realDest int
	tun      *water.Interface
}

type LinkEmulator interface {
	applyEmulation()
	readOutgoingPacket() Packet
	writeIncomingPacket(Packet)
	srcAddr() Address
	dstAddr() Address
}

type DelayEmulator struct {
	inputQueue  chan Packet
	outputQueue chan Packet
	delay       time.Duration
	src         Address
	dst         Address
}

func (e DelayEmulator) applyEmulation() {
	p := <-e.inputQueue
	releaseTime := p.arrivalTime.Add(e.delay)
	delay := releaseTime.Sub(time.Now())
	if delay > 0 {
		time.Sleep(delay)
	}
	e.outputQueue <- p
}

func (e DelayEmulator) writeIncomingPacket(p Packet) {
	e.inputQueue <- p
}

func (e DelayEmulator) readOutgoingPacket() Packet {
	return <-e.outputQueue
}

func (e DelayEmulator) srcAddr() Address {
	return e.src
}

func (e DelayEmulator) dstAddr() Address {
	return e.dst
}

func (s *BroadcastSimulator) processIncomingPackets() {
	for _, emulatorList := range s.queues {
		for _, emulator := range emulatorList {
			go func(e LinkEmulator) {
				for {
					e.applyEmulation()
				}
			}(emulator)
		}
	}
}

func (s *BroadcastSimulator) writeToDestination(p Packet) {
	decodedPacket := gopacket.NewPacket(p.data, layers.IPProtocolIPv4, gopacket.Default)
	if ipLayer := decodedPacket.Layer(layers.LayerTypeIPv4); ipLayer != nil {
		ip, _ := ipLayer.(*layers.IPv4)
		fmt.Printf("From src ip %s to dst ip %s\n", ip.SrcIP.String(), ip.DstIP.String())
		ip.SrcIP = net.IP{10, 0, 0, 1}
		buf := gopacket.NewSerializeBuffer()
		err := ip.SerializeTo(buf, gopacket.SerializeOptions{})
		if err != nil {
			panic(err)
		}
		// TODO(aditi): Find a way to do this that uses the api
		data := append(buf.Bytes(), ip.LayerPayload()...)
		s.tun.Write(data)
	} else {
		panic("unable to decode packet")
	}
}

func (s *BroadcastSimulator) processOutgoingPackets() {
	for src, emulatorList := range s.queues {
		for _, emulator := range emulatorList {
			go func(e LinkEmulator) {
				for {
					packet := e.readOutgoingPacket()
					// If the emulation is complete for the "real dest", we can send it out on the real device
					if e.dstAddr() == s.realDest {
						s.writeToDestination(packet)
					} else if packet.hopsLeft > 0 {
						for _, dstEmulator := range emulatorList {
							if dstEmulator != e {
								packet.src = src
								packet.dst = dstEmulator.dstAddr()
								packet.hopsLeft -= 1
								packet.arrivalTime = time.Now()
								dstEmulator.writeIncomingPacket(packet)
							}
						}
					}
				}
			}(emulator)
		}
	}
}

func main() {
	numAddresses := 4
	realSrc := 0  // Lowest address
	realDest := 3 // Highest address
	maxQueueLength := 1000
	maxHops := 4
	dev, err := water.NewTUN("proxy")

	if err != nil {
		panic(err)
	}

	exec.Command("ip", "link", "set", "dev", dev.Name(), "up").Run()
	exec.Command("ip", "route", "add", "10.0.0.2", "dev", dev.Name()).Run()
	exec.Command("ip", "rule", "add", "from", "100.64.0.1", "table", "1").Run()
	exec.Command("ip", "route", "add", "default", "dev", dev.Name(), "table", "1").Run()

	sim := BroadcastSimulator{make(map[Address]([]LinkEmulator)), realDest, dev}

	// initialize simulator
	for i := 0; i < realDest; i++ {
		sim.queues[i] = make([]LinkEmulator, 0)
		for j := 0; j < numAddresses; j++ {
			if i != j {
				emulator := DelayEmulator{make(chan Packet, maxQueueLength), make(chan Packet, maxQueueLength), time.Millisecond * 10, i, j}
				sim.queues[i] = append(sim.queues[i], emulator)
			}
		}
	}

	// start simulator
	sim.processIncomingPackets()
	sim.processOutgoingPackets()

	packet := make([]byte, 2000)
	for {
		n, err := dev.Read(packet)
		if err != nil {
			panic(err)
		}

		fmt.Printf("packet in %d\n", n)

		for _, emulator := range sim.queues[0] {
			emulator.writeIncomingPacket(Packet{realSrc, emulator.srcAddr(), maxHops, packet[:n], time.Now()})
		}
	}

}
