package simulation

import (
	"time"
)

type SelfState struct {
	latestLatency time.Duration
	latestArrival time.Time
}

type BestNeighborSimulator struct {
	// For each drone maintain map to measurement information for all neighbors
	selfState map[Address]SelfState
	neighbors map[Address][]Address
	realDest  int
}

func NewBestNeighborSimulator(linkConfigs []LinkConfig, realDest Address) *BestNeighborSimulator {
	selfState := make(map[Address]SelfState)
	neighbors := make(map[Address][]Address)
	for _, linkConfig := range linkConfigs {
		srcAddr := linkConfig.SrcAddr()
		dstAddr := linkConfig.DstAddr()
		if _, ok := selfState[srcAddr]; !ok {
			selfState[srcAddr] = SelfState{latestLatency: 0}
		}
		if _, ok := selfState[dstAddr]; !ok {
			selfState[dstAddr] = SelfState{latestLatency: 0}
		}
		if _, ok := neighbors[srcAddr]; !ok {
			neighbors[srcAddr] = make([]Address, 0)
		}
		if dstAddr != realDest {
			srcNeighbors := neighbors[srcAddr]
			srcNeighbors = append(srcNeighbors, dstAddr)
			neighbors[srcAddr] = srcNeighbors
		}
	}
	return &BestNeighborSimulator{
		selfState: selfState,
		neighbors: neighbors,
		realDest:  realDest,
	}
}

func (s *BestNeighborSimulator) OnIncomingPacket(src Address, dst Address) {
	if dst == s.realDest {
		state := s.selfState[src]
		state.latestArrival = time.Now()
		s.selfState[src] = state
	}
}

func (s *BestNeighborSimulator) OnOutgoingPacket(src Address, dst Address) {
	if dst == s.realDest {
		state := s.selfState[src]
		state.latestLatency = time.Since(state.latestArrival)
		s.selfState[src] = state
	}
}

func (s *BestNeighborSimulator) RouteTo(packet Packet, outgoingAddr Address) []Address {
	// TODO(aditi): replace with better version of max int
	lowestLatency := 10000 * time.Second
	bestNeighbor := -1
	for _, addr := range s.neighbors[outgoingAddr] {
		measurement := s.selfState[addr]
		if lowestLatency == -1 || measurement.latestLatency < lowestLatency {
			lowestLatency = measurement.latestLatency
			bestNeighbor = addr
		}
	}
	dests := []Address{s.realDest}
	if bestNeighbor != -1 {
		dests = append(dests, bestNeighbor)
	}
	return dests
}
