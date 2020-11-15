package simulation

import (
	"fmt"
	"net"
	"sync"
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
	queues        map[Address](map[Address]LinkEmulator) // Map src to list of links
	realDest      int
	tun           *water.Interface
	tunDest       net.IP
	router        RoutingSimulator
	sentPackets   map[int]bool
	unsentPackets map[int]string
	mutex         sync.Mutex
}

func NewSimulator(baseAddress Address, device *water.Interface, deviceDstAddr net.IP) BaseSimulator {
	return BaseSimulator{
		queues:        make(map[Address](map[Address]LinkEmulator)),
		realDest:      baseAddress,
		tun:           device,
		tunDest:       deviceDstAddr,
		sentPackets:   make(map[int]bool),
		unsentPackets: make(map[int]string),
	}
}

func (s *BaseSimulator) SetRouter(rs RoutingSimulator) {
	s.router = rs
}

func (s *BaseSimulator) Start(linkConfigs []LinkConfig, maxQueueLength int) {
	log.WithFields(log.Fields{
		"event": "start_simulator",
	}).Info()
	packetLock := sync.Mutex{}
	for _, linkConfig := range linkConfigs {
		srcAddr := linkConfig.SrcAddr()
		if _, ok := s.queues[srcAddr]; !ok {
			s.queues[srcAddr] = make(map[Address]LinkEmulator)
		}
		emu := linkConfig.ToLinkEmulator(maxQueueLength)
		emu.SetOnIncomingPacket(func(link LinkEmulator, p Packet) {
			packetLock.Lock()
			s.router.OnLinkInputDequeue(link, p)
		})
		emu.SetOnOutgoingPacket(func(link LinkEmulator, p Packet) {
			s.router.OnLinkOutputEnqueue(link, p)
			packetLock.Unlock()
		})
		s.queues[srcAddr][linkConfig.DstAddr()] = emu
	}
	s.ProcessIncomingPackets()
	s.ProcessOutgoingPackets()
	go func() {
		for {
			time.Sleep(50 * time.Millisecond)
			log.Error("Unsent packets ", s.unsentPackets)
		}
	}()
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
		} else if tcpLayer := decodedPacket.Layer(layers.LayerTypeTCP); tcpLayer != nil {
			tcp, _ := tcpLayer.(*layers.TCP)
			tcp.SetNetworkLayerForChecksum(ip)
			err := gopacket.SerializeLayers(
				buf,
				gopacket.SerializeOptions{ComputeChecksums: true},
				ip,
				tcp,
				gopacket.Payload(tcpLayer.LayerPayload()),
			)
			if err != nil {
				panic(err)
			}
		} else {
			fmt.Println("All packet layers:")
			for _, layer := range decodedPacket.Layers() {
				fmt.Println("- ", layer.LayerType())
			}
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
	packet.SetSrc(srcAddr)
	packets := s.router.GetRoutedPackets(packet, srcAddr)
	for _, packet := range packets {
		packet.SetArrivalTime(time.Now())
		emulator := s.queues[srcAddr][packet.GetDst()]
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
					// Don't process zero'd out packet
					if e.DstAddr() == s.realDest {
						log.Error(fmt.Sprintf("Arrive packet %d, link (%d, %d), size %d\n", packet.GetId(), e.SrcAddr(), e.DstAddr(), len(packet.GetData())))
					}
					if len(packet.GetData()) > 0 {
						if e.DstAddr() == s.realDest {
							s.writeToDestination(packet)
							log.Error(fmt.Sprintf("Sent packet %d, link(%d, %d)\n", packet.GetId(), e.SrcAddr(), e.DstAddr()))
							s.mutex.Lock()
							log.Error(fmt.Sprintf("Time in transit packet %d, in %s, out %s\n", packet.GetId(), s.unsentPackets[packet.GetId()], time.Now().Format(time.StampMicro)))
							delete(s.unsentPackets, packet.GetId())
							s.sentPackets[packet.GetId()] = true
							s.mutex.Unlock()
						} else if packet.GetHopsLeft() > 0 {
							packet.SetHopsLeft(packet.GetHopsLeft() - 1)
							s.routePacket(packet, e.DstAddr())
						}
					}
				}
			}(emulator)
		}
	}
}

func (s *BaseSimulator) WriteNewPacket(packet Packet, source Address) {
	s.mutex.Lock()
	s.unsentPackets[packet.GetId()] = time.Now().Format(time.StampMicro)
	s.mutex.Unlock()
	s.routePacket(packet, source)
}
