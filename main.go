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

// TODO(aditi): maybe put this in goroutine?
func (e DelayEmulator) applyEmulation() {
	p := <-e.inputQueue
	releaseTime := p.arrivalTime.Add(e.delay)
	delay := releaseTime.Sub(time.Now())
	time.Sleep(delay)
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
		fmt.Printf("From src ip %s to dst ip%s\n", ip.SrcIP.String(), ip.DstIP.String())
		ip.SrcIP = net.IP{10, 0, 0, 1}
		buf := gopacket.NewSerializeBuffer()
		err := ip.SerializeTo(buf, gopacket.SerializeOptions{})
		if err != nil {
			panic(err)
		}
		s.tun.Write(buf.Bytes())
	}
	panic("unable to decode packet")
}

func (s *BroadcastSimulator) processOutgoingPackets() {
	for src, emulatorList := range s.queues {
		for _, emulator := range emulatorList {
			go func(e LinkEmulator) {
				for {
					packet := e.readOutgoingPacket()
					if packet.src == s.realDest {
						s.writeToDestination(packet)
					}
					for _, dstEmulator := range emulatorList {
						if dstEmulator != e {
							packet.src = src
							packet.dst = dstEmulator.dstAddr()
							// TODO(aditi) Not sure if time should be set here
							packet.arrivalTime = time.Now()
							dstEmulator.writeIncomingPacket(packet)
						}
					}
				}
			}(emulator)
		}
	}
}

func main() {
	numAddresses := 4
	realSrc := 0
	realDest := 3
	maxQueueLength := 1000
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
	for i := 1; i < numAddresses; i++ {
		sim.queues[i] = make([]LinkEmulator, numAddresses)
		for j := 1; j < numAddresses; j++ {
			if i != j {
				emulator := DelayEmulator{make(chan Packet, maxQueueLength), make(chan Packet, maxQueueLength), time.Millisecond * 1, i, j}
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
			emulator.writeIncomingPacket(Packet{realSrc, emulator.srcAddr(), packet[:n], time.Now()})
		}
	}

}
