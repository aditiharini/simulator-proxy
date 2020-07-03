package simulation

import (
	"bufio"
	"os"
	"strconv"
	"time"
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
}

func (t TraceEmulator) loadTrace(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		nextSend, err := strconv.Atoi(scanner.Text())
		if err != nil {
			panic(err)
		}
		nextSendOffset := time.Duration(nextSend) * time.Millisecond
		t.sendOffsets = append(t.sendOffsets, nextSendOffset)
	}
}

func (t TraceEmulator) nextReleaseTime() time.Time {
	return t.baseTime.Add(t.sendOffsets[t.currentOffsetIndex])
}

func (t TraceEmulator) skipUnusedSlots(arrivalTime time.Time) {
	for releaseTime := t.nextReleaseTime(); releaseTime.Before(arrivalTime); {
		t.useDeliverySlot()
	}
}

func (t TraceEmulator) useDeliverySlot() {
	t.currentOffsetIndex++
	if t.currentOffsetIndex == len(t.sendOffsets) {
		t.currentOffsetIndex = 0
		t.baseTime = t.baseTime.Add(t.sendOffsets[len(t.sendOffsets)-1])
	}
	t.bytesLeftInDeliveryWindow = 1504
}

func (t TraceEmulator) waitForNextDeliveryOpportunity() {
	releaseTime := t.nextReleaseTime()
	waitTime := releaseTime.Sub(time.Now())
	if waitTime > 0 {
		time.Sleep(waitTime)
	}
}

func (t TraceEmulator) sendPartialPacket() {
	t.havePacketInTransit = false
	t.bytesLeftInDeliveryWindow -= t.bytesLeftInTransit
	t.bytesLeftInTransit = 0
	t.outputQueue <- t.packetInTransit
}

func (t TraceEmulator) sendFullPacket(p Packet) {
	t.bytesLeftInDeliveryWindow -= len(p.Data)
	t.outputQueue <- p
}

func (t TraceEmulator) sendNewPacketsImmediatelyIfPossible() {
	for t.bytesLeftInDeliveryWindow > 0 {
		select {
		case p := <-t.inputQueue:
			if len(p.Data) <= t.bytesLeftInDeliveryWindow {
				t.bytesLeftInDeliveryWindow -= len(p.Data)
				t.outputQueue <- p
			} else {
				t.havePacketInTransit = true
				t.packetInTransit = p
				t.bytesLeftInTransit = len(p.Data) - t.bytesLeftInDeliveryWindow
				t.bytesLeftInDeliveryWindow = 0
			}
		default:
			t.bytesLeftInDeliveryWindow = 0
			return
		}
	}
}

// Whatever simulator is running should be
// calling this function in a loop
func (t TraceEmulator) ApplyEmulation() {
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

	p := <-t.inputQueue
	t.skipUnusedSlots(time.Now())
	t.waitForNextDeliveryOpportunity()
	t.useDeliverySlot()
	t.sendFullPacket(p)
	t.sendNewPacketsImmediatelyIfPossible()
}

func (t TraceEmulator) WriteIncomingPacket(p Packet) {
	t.inputQueue <- p
}

func (t TraceEmulator) ReadOutgoingPacket() Packet {
	return <-t.outputQueue
}
