package simulation

import (
	"fmt"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/songgao/water"
)

type Simulator interface {
	Start(linkConfigs []LinkConfig)
	ProcessOutgoingPackets()
	ProcessIncomingPackets()
	writeToDestination(Packet)
}

type BroadcastSimulator struct {
	queues   map[Address]([](LinkEmulator)) // Map src to list of links
	realDest int
	tun      *water.Interface
	tunDest  net.IP
}

func NewBroadcastSimulator(baseAddress Address, device *water.Interface, deviceDstAddr net.IP) BroadcastSimulator {
	return BroadcastSimulator{
		queues:   make(map[Address]([]LinkEmulator)),
		realDest: baseAddress,
		tun:      device,
		tunDest:  deviceDstAddr,
	}
}

func (s *BroadcastSimulator) Start(linkConfigs []LinkConfig, maxQueueLength int) {
	for _, linkConfig := range linkConfigs {
		srcAddr := linkConfig.SrcAddr()
		if _, ok := s.queues[srcAddr]; !ok {
			s.queues[srcAddr] = make([]LinkEmulator, 0)
		}
		s.queues[srcAddr] = append(s.queues[srcAddr], linkConfig.ToLinkEmulator(maxQueueLength))
	}
	s.ProcessIncomingPackets()
	s.ProcessOutgoingPackets()
}

func (s *BroadcastSimulator) ProcessIncomingPackets() {
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

// TODO(aditi): This is pretty heavyweight.
func (s *BroadcastSimulator) writeToDestination(p Packet) {
	decodedPacket := gopacket.NewPacket(p.Data, layers.IPProtocolIPv4, gopacket.Default)
	if ipLayer := decodedPacket.Layer(layers.LayerTypeIPv4); ipLayer != nil {
		ip, _ := ipLayer.(*layers.IPv4)
		ip.SrcIP = s.tunDest
		buf := gopacket.NewSerializeBuffer()
		if udpLayer := decodedPacket.Layer(layers.LayerTypeUDP); udpLayer != nil {
			udp, _ := udpLayer.(*layers.UDP)
			udp.SetNetworkLayerForChecksum(ip)
			err := gopacket.SerializeLayers(
				buf,
				gopacket.SerializeOptions{ComputeChecksums: true},
				ip,
				udp,
				gopacket.Payload(udpLayer.LayerPayload()),
			)
			if err != nil {
				panic(err)
			}
		} else if tcpLayer := decodedPacket.Layer(layers.LayerTypeUDP); udpLayer != nil {
			tcp, _ := tcpLayer.(*layers.UDP)
			tcp.SetNetworkLayerForChecksum(ip)
			err := gopacket.SerializeLayers(
				buf,
				gopacket.SerializeOptions{ComputeChecksums: true},
				ip,
				tcp,
				gopacket.Payload(udpLayer.LayerPayload()),
			)
			if err != nil {
				panic(err)
			}
		} else {
			panic("unsupported application layer")
		}
		fmt.Printf("packet %v from %v\n", p.Id, p.Src)
		// TODO(aditi): Find a way to do this that uses the api
		s.tun.Write(buf.Bytes())
	} else {
		panic("unable to decode packet")
	}
}

func (s *BroadcastSimulator) ProcessOutgoingPackets() {
	for _, emulatorList := range s.queues {
		for _, emulator := range emulatorList {
			go func(e LinkEmulator) {
				for {
					packet := e.ReadOutgoingPacket()
					// If the emulation is complete for the "real dest", we can send it out on the real device
					if e.DstAddr() == s.realDest {
						s.writeToDestination(packet)
					} else if packet.HopsLeft > 0 {
						s.BroadcastPacket(packet, e.DstAddr())
					}
				}
			}(emulator)
		}
	}
}

func (s *BroadcastSimulator) BroadcastPacket(packet Packet, outgoingAddr Address) {
	hopsLeft := packet.HopsLeft - 1
	packet.Src = outgoingAddr
	for _, dstEmulator := range s.queues[outgoingAddr] {
		packet.Dst = dstEmulator.DstAddr()
		packet.HopsLeft = hopsLeft
		packet.ArrivalTime = time.Now()
		dstEmulator.WriteIncomingPacket(packet)
	}
}
