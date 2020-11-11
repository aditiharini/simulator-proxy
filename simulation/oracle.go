package simulation

import "sync"

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
}

func NewOracleRouter(neighbors NeighborMap) *OracleRouter {
	return &OracleRouter{neighbors: neighbors, seenPackets: make(map[int]int)}
}

func (or *OracleRouter) OnLinkDequeue(p Packet) {
	switch v := p.(type) {
	case *OraclePacket:
		// do something
		if _, ok := or.seenPackets[v.GetId()]; ok {
			v.ClearData()
			or.seenPackets[v.GetId()]++
		} else {
			or.seenPackets[v.GetId()] = 1
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

func (or *OracleRouter) GetRoutedPackets(packet Packet, outgoingAddr Address) []Packet {
	var packets []Packet
	for _, neighbor := range or.neighbors[outgoingAddr] {
		newPacket := packet.Copy()
		newPacket.SetDst(neighbor)
		switch v := newPacket.(type) {
		case *OraclePacket:
			packets = append(packets, newPacket)
		case *DataPacket:
			// Need to deepcopy packets
			packets = append(packets, &OraclePacket{*v})
		}
	}
	return packets
}
