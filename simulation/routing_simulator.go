package simulation

type DroneInfo struct {
	latestLatency   int
	latestBandwidth int
}

type DroneState struct {
	neighborState map[Address]DroneInfo
}

type MeasurementPacket struct {
	Packet
	measurement DroneInfo
}

func RoutePacket(p Packet) {

}
