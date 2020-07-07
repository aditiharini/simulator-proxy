package simulation

import (
	"fmt"
	"time"
)

type DelayEmulator struct {
	inputQueue  chan Packet
	outputQueue chan Packet
	delay       time.Duration
	src         Address
	dst         Address
}

func NewDelayEmulator(maxQueueLength int, delay time.Duration, src Address, dst Address) DelayEmulator {
	return DelayEmulator{
		inputQueue:  make(chan Packet, maxQueueLength),
		outputQueue: make(chan Packet, maxQueueLength),
		delay:       delay,
		src:         src,
		dst:         dst}
}

func (e *DelayEmulator) ApplyEmulation() {
	p := <-e.inputQueue
	releaseTime := p.ArrivalTime.Add(e.delay)
	delay := releaseTime.Sub(time.Now())
	if delay > 0 {
		time.Sleep(delay)
	}
	e.outputQueue <- p
}

func (e *DelayEmulator) WriteIncomingPacket(p Packet) {
	fmt.Printf("emulator %v %v received packet\n", e.src, e.dst)
	e.inputQueue <- p
}

func (e *DelayEmulator) ReadOutgoingPacket() Packet {
	return <-e.outputQueue
}

func (e *DelayEmulator) SrcAddr() Address {
	return e.src
}

func (e *DelayEmulator) DstAddr() Address {
	return e.dst
}
