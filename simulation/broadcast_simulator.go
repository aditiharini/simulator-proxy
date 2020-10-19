package simulation

type BroadcastSimulator struct {
	neighbors map[Address][]Address
}

func NewBroadcastSimulator(linkConfigs []LinkConfig) *BroadcastSimulator {
	neighborMap := make(map[Address][]Address)
	for _, linkConfig := range linkConfigs {
		srcAddr := linkConfig.SrcAddr()
		if _, ok := neighborMap[srcAddr]; !ok {
			neighborMap[srcAddr] = make([]Address, 0)
		}
		neighbors := neighborMap[srcAddr]
		neighborMap[srcAddr] = append(neighbors, linkConfig.DstAddr())
	}
	return &BroadcastSimulator{neighbors: neighborMap}
}

func (s *BroadcastSimulator) OnIncomingPacket(src Address, dst Address) {
	// Do nothing
	return
}

func (s *BroadcastSimulator) OnOutgoingPacket(src Address, dst Address) {
	// Do nothing
	return
}

func (s *BroadcastSimulator) RouteTo(packet Packet, outgoingAddr Address) []Address {
	return s.neighbors[outgoingAddr]
}
