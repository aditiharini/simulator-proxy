package simulation

import "time"

type LinkConfig interface {
	ToLinkEmulator(queueSize int) LinkEmulator
	SrcAddr() Address
	DstAddr() Address
}

type NeighborMap = map[Address][]Address

func ToNeighborsMap(configs []LinkConfig) NeighborMap {
	neighborMap := make(map[Address][]Address)
	for _, config := range configs {
		src := config.SrcAddr()
		dst := config.DstAddr()
		if _, ok := neighborMap[src]; !ok {
			neighborMap[src] = make([]Address, 0)
		}
		neighbors := neighborMap[src]
		neighborMap[src] = append(neighbors, dst)
	}
	return neighborMap
}

type DelayLinkConfig struct {
	delay time.Duration
	src   Address
	dst   Address
}

func NewDelayLinkConfig(delay time.Duration, src Address, dst Address) DelayLinkConfig {
	return DelayLinkConfig{
		delay,
		src,
		dst,
	}
}

func (c DelayLinkConfig) ToLinkEmulator(queueSize int) LinkEmulator {
	// TODO(aditi): make NewDelayEmulator return a pointer
	newEmulator := NewDelayEmulator(queueSize, c.delay, c.src, c.dst)
	return &newEmulator
}

func (c DelayLinkConfig) SrcAddr() Address {
	return c.src
}

func (c DelayLinkConfig) DstAddr() Address {
	return c.dst
}

type TraceLinkConfig struct {
	filename     string
	lossfilename string
	src          Address
	dst          Address
}

func NewTraceLinkConfig(filename string, lossfilename string, src Address, dst Address) TraceLinkConfig {
	return TraceLinkConfig{
		filename,
		lossfilename,
		src,
		dst,
	}
}

func (c TraceLinkConfig) ToLinkEmulator(queueSize int) LinkEmulator {
	// TODO(aditi): make NewTraceEmulator return a pointer too
	newEmulator := NewTraceEmulator(c.filename, c.lossfilename, queueSize, c.src, c.dst)
	return &newEmulator
}

func (c TraceLinkConfig) SrcAddr() Address {
	return c.src
}

func (c TraceLinkConfig) DstAddr() Address {
	return c.dst
}
