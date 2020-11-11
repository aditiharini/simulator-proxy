package simulation

type BroadcastSimulator struct {
	neighbors NeighborMap
}

func NewBroadcastSimulator(neighbors NeighborMap) *BroadcastSimulator {
	return &BroadcastSimulator{neighbors: neighbors}
}

func (s *BroadcastSimulator) OnLinkDequeue(p Packet) {
	// Do nothing
	return
}

func (s *BroadcastSimulator) OnIncomingPacket(src Address, dst Address) {
	// Do nothing
	return
}

func (s *BroadcastSimulator) OnOutgoingPacket(p Packet) {
	// Do nothing
	return
}

func (s *BroadcastSimulator) GetRoutedPackets(packet Packet, outgoingAddr Address) []Packet {
	var packets []Packet
	for _, neighbor := range s.neighbors[outgoingAddr] {
		newPacket := packet.Copy()
		newPacket.SetDst(neighbor)
		packets = append(packets, newPacket)
	}
	return packets
}
