package simulation

import (
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"
)

type OraclePacket struct {
	DataPacket
}

func (op *OraclePacket) Copy() Packet {
	packet := *op
	newPacket := packet
	return &newPacket
}

type OracleRouter struct {
	seenPackets map[int]int
	mutex       sync.Mutex
	neighbors   NeighborMap
	realDest    Address
}

func NewOracleRouter(neighbors NeighborMap, realDest Address) *OracleRouter {
	return &OracleRouter{neighbors: neighbors, seenPackets: make(map[int]int), realDest: realDest}
}

func (or *OracleRouter) OnLinkInputDequeue(link LinkEmulator, p Packet) {
	switch v := p.(type) {
	case *OraclePacket:
		// do something
		if link.DstAddr() == or.realDest {
			or.mutex.Lock()
			if _, ok := or.seenPackets[v.GetId()]; ok {
				v.ClearData()
				log.Error(fmt.Sprintf("Filter packet %d (%d), link (%d, %d), size %d\n", p.GetId(), or.seenPackets[v.GetId()], link.SrcAddr(), link.DstAddr(), len(p.GetData())))
				or.seenPackets[v.GetId()]++
			}
			or.mutex.Unlock()
		}
	default:
		// do nothing
	}
}

func (or *OracleRouter) OnLinkOutputEnqueue(link LinkEmulator, p Packet) {
	switch v := p.(type) {
	case *OraclePacket:
		// do something
		if link.DstAddr() == or.realDest {
			or.mutex.Lock()
			if _, ok := or.seenPackets[v.GetId()]; !ok {
				or.seenPackets[v.GetId()] = 1
				log.Error(fmt.Sprintf("Keep packet %d, link (%d, %d)\n", p.GetId(), link.SrcAddr(), link.DstAddr()))
			}
			or.mutex.Unlock()
		}
	default:
		// do nothing
	}
}

func (or *OracleRouter) OnIncomingPacket(src Address, dst Address) {
	// Do nothing
}

func (or *OracleRouter) OnOutgoingPacket(p Packet) {
	// Do nothing
}

func (or *OracleRouter) PrintState() {
	fmt.Println("NUM SEEN", len(or.seenPackets))
}

func (or *OracleRouter) GetRoutedPackets(packet Packet, outgoingAddr Address) []Packet {
	var packets []Packet
	for _, neighbor := range or.neighbors[outgoingAddr] {
		newPacket := packet.Copy()
		newPacket.SetDst(neighbor)
		switch v := newPacket.(type) {
		case *OraclePacket:
			packets = append(packets, newPacket)
		case *DataPacket:
			packets = append(packets, &OraclePacket{*v})
		}
	}
	return packets
}
