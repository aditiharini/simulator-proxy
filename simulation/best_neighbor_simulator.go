package simulation

import (
	"time"
)

type NeighborState struct {
	latestLatency int
}

type SelfState struct {
	latestLatency time.Duration
	latestArrival time.Time
}

type BestNeighborSimulator struct {
	// For each drone maintain map to measurement information for all neighbors
	neighborState map[Address]map[Address]NeighborState
	selfState     map[Address]SelfState
}

type MeasurementPacket struct {
	Packet
	measurement SelfState
}

func NewBestNeighborSimulator(linkConfigs []LinkConfig) BestNeighborSimulator {
	neighborState := make(map[Address]map[Address]NeighborState)
	selfState := make(map[Address]SelfState)
	for _, linkConfig := range linkConfigs {
		srcAddr := linkConfig.SrcAddr()
		dstAddr := linkConfig.DstAddr()
		// TODO(aditi): Add empty check
		neighborState[srcAddr][dstAddr] = NeighborState{}
		selfState[srcAddr] = SelfState{}
		selfState[dstAddr] = SelfState{}
	}
	return BestNeighborSimulator{
		neighborState: neighborState,
		selfState:     selfState,
	}
}

func (s *BestNeighborSimulator) OnIncomingPacket(src Address, dst Address) {
}

func (s *BestNeighborSimulator) OnOutgoingPacket(src Address, dst Address) {

}

func (s *BestNeighborSimulator) RouteTo(packet Packet, outgoingAddr Address) []Address {
	lowestLatency := -1
	bestNeighbor := -1
	for neighbor, measurement := range s.neighborState[outgoingAddr] {
		if lowestLatency == -1 || measurement.latestLatency > lowestLatency {
			lowestLatency = measurement.latestLatency
			bestNeighbor = neighbor
		}
	}
	return []Address{bestNeighbor}
}
