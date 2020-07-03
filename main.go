package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"

	. "github.com/aditiharini/simulator-proxy/simulation"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/songgao/water"
)

type Simulator interface {
	processOutgoingPackets()
	processIncomingPackets()
	writeToDestination(Packet)
}

type BroadcastSimulator struct {
	queues   map[Address]([]LinkEmulator) // Map src to list of links
	realDest int
	tun      *water.Interface
	tunDest  net.IP
}

func (s *BroadcastSimulator) processIncomingPackets() {
	for _, emulatorList := range s.queues {
		for _, emulator := range emulatorList {
			go func(e LinkEmulator) {
				for {
					// Apply emulation to packet as soon as it's received
					e.ApplyEmulation()
				}
			}(emulator)
		}
	}
}

func (s *BroadcastSimulator) writeToDestination(p Packet) {
	decodedPacket := gopacket.NewPacket(p.Data, layers.IPProtocolIPv4, gopacket.Default)
	if ipLayer := decodedPacket.Layer(layers.LayerTypeIPv4); ipLayer != nil {
		ip, _ := ipLayer.(*layers.IPv4)
		fmt.Printf("From src ip %s to dst ip %s\n", ip.SrcIP.String(), ip.DstIP.String())
		ip.SrcIP = s.tunDest
		buf := gopacket.NewSerializeBuffer()
		err := ip.SerializeTo(buf, gopacket.SerializeOptions{ComputeChecksums: true})
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
					packet := e.ReadOutgoingPacket()
					// If the emulation is complete for the "real dest", we can send it out on the real device
					if e.DstAddr() == s.realDest {
						s.writeToDestination(packet)
					} else if packet.HopsLeft > 0 {
						for _, dstEmulator := range emulatorList {
							if dstEmulator != e {
								packet.Src = src
								packet.Dst = dstEmulator.DstAddr()
								packet.HopsLeft -= 1
								packet.ArrivalTime = time.Now()
								dstEmulator.WriteIncomingPacket(packet)
							}
						}
					}
				}
			}(emulator)
		}
	}
}

func readConfig(filename string) Config {
	var config Config
	confFile, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer confFile.Close()
	err = json.NewDecoder(confFile).Decode(&config)
	if err != nil {
		panic(err)
	}
	return config
}

type Config struct {
	NumAddresses        int    `json:"numAddresses"`
	RealSrcAddress      string `json:"realSrcAddress"`
	SimulatedSrcAddress int    `json:"simulatedSrcAddress"`
	SimulatedDstAddress int    `json:"simulatedDstAddress"`
	MaxQueueLength      int    `json:"maxQueueLength"`
	MaxHops             int    `json:"maxHops"`
	DevName             string `json:"devName"`
	DevSrcAddr          string `json:"devSrcAddr"`
	DevDstAddr          string `json:"devDstAddr"`
	RoutingTableNum     string `json:"routingTableNum"`
}

func main() {
	config := readConfig("./config.json")
	dev, err := water.NewTUN(config.DevName)

	if err != nil {
		panic(err)
	}

	exec.Command("ip", "rule", "delete", "table", config.RoutingTableNum).Run()
	exec.Command("ip", "link", "set", "dev", dev.Name(), "up").Run()
	exec.Command("ip", "addr", "add", config.DevSrcAddr, "dev", dev.Name()).Run()
	exec.Command("ip", "rule", "add", "from", config.RealSrcAddress, "table", "1").Run()
	exec.Command("ifconfig", dev.Name(), config.DevSrcAddr, "dstaddr", config.DevDstAddr).Run()
	exec.Command("ip", "route", "add", "default", "dev", dev.Name(), "table", config.RoutingTableNum).Run()

	sim := BroadcastSimulator{make(map[Address]([]LinkEmulator)), config.SimulatedDstAddress, dev, net.ParseIP(config.DevDstAddr)}

	// initialize simulator
	for i := 0; i < config.SimulatedDstAddress; i++ {
		sim.queues[i] = make([]LinkEmulator, 0)
		for j := 0; j < config.NumAddresses; j++ {
			if i != j {
				emulator := NewDelayEmulator(make(chan Packet, config.MaxQueueLength), make(chan Packet, config.MaxQueueLength), time.Millisecond*10, i, j)
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
			emulator.WriteIncomingPacket(Packet{
				Src:         config.SimulatedSrcAddress,
				Dst:         emulator.SrcAddr(),
				HopsLeft:    config.MaxHops,
				Data:        packet[:n],
				ArrivalTime: time.Now(),
			})
		}
	}

}
