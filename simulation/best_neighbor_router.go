package simulation

import (
	"sync"
	"time"
)

type SelfState struct {
	latestLatency time.Duration
	latestArrival time.Time
}

type BestNeighborSimulator struct {
	// For each drone maintain map to measurement information for all neighbors
	selfState       map[Address]SelfState
	neighbors       NeighborMap
	realDest        int
	updateLagMillis time.Duration
	mutex           sync.Mutex
}

func NewBestNeighborSimulator(neighborMap NeighborMap, realDest Address, updateLagMillis time.Duration) *BestNeighborSimulator {
	selfState := make(map[Address]SelfState)
	for src, neighbors := range neighborMap {
		for _, dst := range neighbors {
			if _, ok := selfState[src]; !ok {
				selfState[src] = SelfState{latestLatency: 0}
			}
			if _, ok := selfState[dst]; !ok {
				selfState[dst] = SelfState{latestLatency: 0}
			}
		}
	}
	return &BestNeighborSimulator{
		selfState: selfState,
		neighbors: neighborMap,
		realDest:  realDest,
	}
}

func (s *BestNeighborSimulator) OnLinkInputDequeue(link LinkEmulator, p Packet) {
	// Do nothing
	return
}

func (s *BestNeighborSimulator) OnLinkOutputEnqueue(link LinkEmulator, p Packet) {
	// Do nothing
	return
}

func (s *BestNeighborSimulator) OnIncomingPacket(src Address, dst Address) {
	if dst == s.realDest {
		state := s.selfState[src]
		state.latestArrival = time.Now()
		s.selfState[src] = state
	}
}

func (s *BestNeighborSimulator) OnOutgoingPacket(p Packet) {
	if p.GetDst() == s.realDest {
		state := s.selfState[p.GetSrc()]
		state.latestLatency = time.Since(state.latestArrival)
		go func(Packet, SelfState) {
			time.Sleep(s.updateLagMillis)
			s.mutex.Lock()
			s.selfState[p.GetSrc()] = state
			s.mutex.Unlock()
		}(p, state)
	}
}

func (s *BestNeighborSimulator) GetRoutedPackets(packet Packet, outgoingAddr Address) []Packet {
	lowestLatency := 10000 * time.Second
	bestNeighbor := -1
	for _, addr := range s.neighbors[outgoingAddr] {
		measurement := s.selfState[addr]
		if addr != s.realDest && (lowestLatency == -1 || measurement.latestLatency < lowestLatency) {
			lowestLatency = measurement.latestLatency
			bestNeighbor = addr
		}
	}
	packet.SetDst(s.realDest)
	packets := []Packet{packet}
	if bestNeighbor != -1 {
		newPacket := packet.Copy()
		newPacket.SetDst(bestNeighbor)
		packets = append(packets, newPacket)
	}
	return packets
}

func (s *BestNeighborSimulator) PrintState() {

}
