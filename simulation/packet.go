package simulation

import "time"

type Address = int

// Enables support for sending different types of packets within the simulator
type Packet interface {
	GetSrc() Address
	SetSrc(addr Address)
	GetDst() Address
	SetDst(addr Address)
	GetHopsLeft() int
	SetHopsLeft(hops int)
	GetData() []byte
	GetArrivalTime() time.Time
	SetArrivalTime(t time.Time)
	GetId() int
	ClearData()
	Copy() Packet
}

type DataPacket struct {
	Src         Address
	Dst         Address
	HopsLeft    int
	Data        []byte
	ArrivalTime time.Time
	Id          int
}

func (dp *DataPacket) GetSrc() Address {
	return dp.Src
}

func (dp *DataPacket) SetSrc(addr Address) {
	dp.Src = addr
}

func (dp *DataPacket) GetDst() Address {
	return dp.Dst
}

func (dp *DataPacket) SetDst(addr Address) {
	dp.Dst = addr
}

func (dp *DataPacket) GetHopsLeft() int {
	return dp.HopsLeft
}

func (dp *DataPacket) SetHopsLeft(hops int) {
	dp.HopsLeft = hops
}

func (dp *DataPacket) GetData() []byte {
	return dp.Data
}

func (dp *DataPacket) GetArrivalTime() time.Time {
	return dp.ArrivalTime
}

func (dp *DataPacket) SetArrivalTime(t time.Time) {
	dp.ArrivalTime = t
}

func (dp *DataPacket) GetId() int {
	return dp.Id
}

func (dp *DataPacket) ClearData() {
	dp.Data = nil
}

func (dp *DataPacket) Copy() Packet {
	origPacket := *dp
	newPacket := origPacket
	return &newPacket
}

type LinkEmulator interface {
	ApplyEmulation()
	ReadOutgoingPacket() Packet
	WriteIncomingPacket(Packet)
	SetOnIncomingPacket(func(LinkEmulator, Packet))
	SetOnOutgoingPacket(func(LinkEmulator, Packet))
	SrcAddr() Address
	DstAddr() Address
}

type OutgoingPacketResponse struct {
	packetsToSend []Packet
}

type RoutingSimulator interface {
	OnIncomingPacket(src Address, dst Address)
	OnOutgoingPacket(p Packet)
	OnLinkInputDequeue(link LinkEmulator, p Packet)
	OnLinkOutputEnqueue(link LinkEmulator, p Packet)
	GetRoutedPackets(packet Packet, outgoingAddr Address) []Packet
	PrintState()
}
