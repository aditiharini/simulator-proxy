package simulation

import (
	"time"
)

type DelayEmulator struct {
	inputQueue             chan Packet
	outputQueue            chan Packet
	delay                  time.Duration
	src                    Address
	dst                    Address
	incomingPacketCallback func(Packet)
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
	p := e.readIncomingPacket()
	releaseTime := p.GetArrivalTime().Add(e.delay)
	delay := releaseTime.Sub(time.Now())
	if delay > 0 {
		time.Sleep(delay)
	}
	e.outputQueue <- p
}

func (e *DelayEmulator) SetOnIncomingPacket(callback func(Packet)) {
	e.incomingPacketCallback = callback
}

func (e *DelayEmulator) readIncomingPacket() Packet {
	p := <-e.inputQueue
	e.incomingPacketCallback(p)
	return p
}

func (e *DelayEmulator) WriteIncomingPacket(p Packet) {
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
