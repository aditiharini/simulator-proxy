package simulation

import "time"

type Address = int

type Packet struct {
	Src         Address
	Dst         Address
	HopsLeft    int
	Data        []byte
	ArrivalTime time.Time
	Id          int
}

type LinkEmulator interface {
	ApplyEmulation()
	ReadOutgoingPacket() Packet
	WriteIncomingPacket(Packet)
	SrcAddr() Address
	DstAddr() Address
}

type RoutingSimulator interface {
	OnIncomingPacket(src Address, dst Address)
	OnOutgoingPacket(src Address, dst Address)
	RouteTo(packet Packet, outgoingAddr Address) []Address
}
