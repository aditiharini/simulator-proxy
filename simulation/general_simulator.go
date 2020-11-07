package simulation

import (
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	log "github.com/sirupsen/logrus"
	"github.com/songgao/water"
)

type Simulator interface {
	Start(linkConfigs []LinkConfig)
	ProcessOutgoingPackets()
	ProcessIncomingPackets()
	writeToDestination(Packet)
}

type BaseSimulator struct {
	queues   map[Address](map[Address]LinkEmulator) // Map src to list of links
	realDest int
	tun      *water.Interface
	tunDest  net.IP
	router   RoutingSimulator
}

func NewSimulator(baseAddress Address, device *water.Interface, deviceDstAddr net.IP) BaseSimulator {
	return BaseSimulator{
		queues:   make(map[Address](map[Address]LinkEmulator)),
		realDest: baseAddress,
		tun:      device,
		tunDest:  deviceDstAddr,
	}
}

func (s *BaseSimulator) SetRouter(rs RoutingSimulator) {
	s.router = rs
}

func (s *BaseSimulator) Start(linkConfigs []LinkConfig, maxQueueLength int) {
	log.WithFields(log.Fields{
		"event": "start_simulator",
	}).Info()
	for _, linkConfig := range linkConfigs {
		srcAddr := linkConfig.SrcAddr()
		if _, ok := s.queues[srcAddr]; !ok {
			s.queues[srcAddr] = make(map[Address]LinkEmulator)
		}
		s.queues[srcAddr][linkConfig.DstAddr()] = linkConfig.ToLinkEmulator(maxQueueLength)
	}
	s.ProcessIncomingPackets()
	s.ProcessOutgoingPackets()
}

func (s *BaseSimulator) ProcessIncomingPackets() {
	for _, emulatorList := range s.queues {
		for _, emulator := range emulatorList {
			go func(e LinkEmulator) {
				for {
					// Apply emulation to packet as soon as it's received
					s.router.OnIncomingPacket(e.SrcAddr(), e.DstAddr())
					e.ApplyEmulation()
				}
			}(emulator)
		}
	}
}

// TODO(aditi): This is pretty heavyweight.
func (s *BaseSimulator) writeToDestination(p Packet) {
	decodedPacket := gopacket.NewPacket(p.GetData(), layers.IPProtocolIPv4, gopacket.Default)
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

		log.WithFields(log.Fields{
			"event": "packet_sent",
			"id":    p.GetId(),
			"src":   p.GetSrc(),
		}).Info()

		// TODO(aditi): Find a way to do this that uses the api
		s.tun.Write(buf.Bytes())
	} else {
		panic("unable to decode packet")
	}
}

func (s *BaseSimulator) routePacket(packet Packet, srcAddr Address) {
	dstAddrs := s.router.RouteTo(packet, srcAddr)
	packet.SetSrc(srcAddr)
	for _, dstAddr := range dstAddrs {
		packet.SetDst(dstAddr)
		packet.SetArrivalTime(time.Now())
		emulator := s.queues[srcAddr][dstAddr]
		emulator.WriteIncomingPacket(packet)
	}
}

func (s *BaseSimulator) ProcessOutgoingPackets() {
	for _, emulatorList := range s.queues {
		for _, emulator := range emulatorList {
			go func(e LinkEmulator) {
				for {
					packet := e.ReadOutgoingPacket()
					s.router.OnOutgoingPacket(packet)
					// If the emulation is complete for the "real dest", we can send it out on the real device
					if e.DstAddr() == s.realDest {
						s.writeToDestination(packet)
					} else if packet.GetHopsLeft() > 0 {
						packet.SetHopsLeft(packet.GetHopsLeft() - 1)
						s.routePacket(packet, e.DstAddr())
					}
				}
			}(emulator)
		}
	}
}

func (s *BaseSimulator) WriteNewPacket(packet Packet, source Address) {
	s.routePacket(packet, source)
}
