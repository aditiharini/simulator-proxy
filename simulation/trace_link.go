package simulation

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

type TraceEmulator struct {
	baseTime                  time.Time
	sendOffsets               []time.Duration
	currentOffsetIndex        int
	inputQueue                chan Packet
	outputQueue               chan Packet
	havePacketInTransit       bool
	packetInTransit           Packet
	bytesLeftInTransit        int
	bytesLeftInDeliveryWindow int
	src                       Address
	dst                       Address
	incomingPacketCallback    func(LinkEmulator, Packet)
	outgoingPacketCallback    func(LinkEmulator, Packet)
}

func (t *TraceEmulator) SrcAddr() Address {
	return t.src
}

func (t *TraceEmulator) DstAddr() Address {
	return t.dst
}

func loadTrace(filename string) []time.Duration {
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var sendOffsets []time.Duration
	for scanner.Scan() {
		nextSend, err := strconv.Atoi(scanner.Text())
		if err != nil {
			panic(err)
		}
		nextSendOffset := time.Duration(nextSend) * time.Millisecond
		sendOffsets = append(sendOffsets, nextSendOffset)
	}
	return sendOffsets
}

func NewTraceEmulator(filename string, maxQueueSize int, src Address, dst Address) TraceEmulator {
	now := time.Now()
	log.WithFields(log.Fields{
		"event": "start_trace",
		"src":   src,
		"dst":   dst,
	}).WithTime(now).Info()
	return TraceEmulator{
		baseTime:                  now,
		sendOffsets:               loadTrace(filename),
		currentOffsetIndex:        0,
		inputQueue:                make(chan Packet, maxQueueSize),
		outputQueue:               make(chan Packet, maxQueueSize),
		havePacketInTransit:       false,
		packetInTransit:           &DataPacket{},
		bytesLeftInDeliveryWindow: 0,
		bytesLeftInTransit:        0,
		src:                       src,
		dst:                       dst,
	}
}

func (t *TraceEmulator) nextReleaseTime() time.Time {
	return t.baseTime.Add(t.sendOffsets[t.currentOffsetIndex])
}

func (t *TraceEmulator) skipUnusedSlots(arrivalTime time.Time) {
	releaseTime := t.nextReleaseTime()
	for releaseTime.Before(arrivalTime) {
		t.useDeliverySlot()
		releaseTime = t.nextReleaseTime()
	}
}

func (t *TraceEmulator) useDeliverySlot() {
	t.currentOffsetIndex++
	if t.currentOffsetIndex == len(t.sendOffsets) {
		t.currentOffsetIndex = 0
		t.baseTime = t.baseTime.Add(t.sendOffsets[len(t.sendOffsets)-1])
	}
	t.bytesLeftInDeliveryWindow = 1504
}

func (t *TraceEmulator) waitForNextDeliveryOpportunity() {
	releaseTime := t.nextReleaseTime()
	waitTime := releaseTime.Sub(time.Now())
	if waitTime > 0 {
		time.Sleep(waitTime)
	}
}

func (t *TraceEmulator) sendPartialPacket() {
	t.havePacketInTransit = false
	t.bytesLeftInDeliveryWindow -= t.bytesLeftInTransit
	t.bytesLeftInTransit = 0
	t.writeOutgoingPacket(t.packetInTransit)
}

func (t *TraceEmulator) sendFullPacket(p Packet) {
	t.bytesLeftInDeliveryWindow -= len(p.GetData())
	t.writeOutgoingPacket(p)
}

func (t *TraceEmulator) sendNewPacketsImmediatelyIfPossible() {
	for t.bytesLeftInDeliveryWindow > 0 {
		p := t.readIncomingPacketIfAvailable()
		if p == nil {
			t.bytesLeftInDeliveryWindow = 0
			return
		} else {
			if len(p.GetData()) <= t.bytesLeftInDeliveryWindow {
				t.bytesLeftInDeliveryWindow -= len(p.GetData())
				t.writeOutgoingPacket(p)
			} else {
				t.havePacketInTransit = true
				t.packetInTransit = p
				t.bytesLeftInTransit = len(p.GetData()) - t.bytesLeftInDeliveryWindow
				t.bytesLeftInDeliveryWindow = 0
			}
		}
	}
}

// Whatever simulator is running should be
// calling this function in a loop
func (t *TraceEmulator) ApplyEmulation() {
	// If there is a packet in transit,
	// we want to send it as soon as we can
	for t.havePacketInTransit {
		// Can't have leftovers in this packet because
		// packet size has to be <= size of delivery slot
		t.waitForNextDeliveryOpportunity()
		t.useDeliverySlot()
		t.sendPartialPacket()
		t.sendNewPacketsImmediatelyIfPossible()
	}

	p := t.readIncomingPacket()
	t.skipUnusedSlots(time.Now())
	t.waitForNextDeliveryOpportunity()
	t.useDeliverySlot()
	t.sendFullPacket(p)
	t.sendNewPacketsImmediatelyIfPossible()
}

func (t *TraceEmulator) SetOnIncomingPacket(callback func(LinkEmulator, Packet)) {
	t.incomingPacketCallback = callback
}

func (t *TraceEmulator) SetOnOutgoingPacket(callback func(LinkEmulator, Packet)) {
	t.outgoingPacketCallback = callback
}

func (t *TraceEmulator) onIncomingPacket(p Packet) {
	t.incomingPacketCallback(t, p)
}

func (t *TraceEmulator) readIncomingPacket() Packet {
	p := <-t.inputQueue
	t.onIncomingPacket(p)
	return p
}

func (t *TraceEmulator) readIncomingPacketIfAvailable() Packet {
	select {
	case p := <-t.inputQueue:
		t.onIncomingPacket(p)
		return p
	default:
		return nil
	}
}

func (t *TraceEmulator) writeOutgoingPacket(p Packet) {
	t.outgoingPacketCallback(t, p)
	t.outputQueue <- p
	log.Error(fmt.Sprintf("Link output packet %d, link (%d, %d), size %d\n", p.GetId(), t.src, t.dst, len(p.GetData())))
}

func (t *TraceEmulator) WriteIncomingPacket(p Packet) {
	log.WithFields(log.Fields{
		"event": "packet_entered_link",
		"id":    p.GetId(),
		"src":   t.src,
		"dst":   t.dst,
	}).Info()
	t.inputQueue <- p
}

func (t *TraceEmulator) ReadOutgoingPacket() Packet {
	p := <-t.outputQueue
	log.WithFields(log.Fields{
		"event": "packet_left_link",
		"id":    p.GetId(),
		"src":   t.src,
		"dst":   t.dst,
	}).Info()
	return p
}
