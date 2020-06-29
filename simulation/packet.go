package simulation

import "time"

type Address = int

type Packet struct {
	Src         Address
	Dst         Address
	HopsLeft    int
	Data        []byte
	ArrivalTime time.Time
}

type LinkEmulator interface {
	ApplyEmulation()
	ReadOutgoingPacket() Packet
	WriteIncomingPacket(Packet)
	SrcAddr() Address
	DstAddr() Address
}
