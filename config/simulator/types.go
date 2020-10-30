package config

// TODO(aditi): Make this easier to change. This is rigid and ugly
type TopologyJson = map[string](map[string]interface{})

type GeneralConfig struct {
	RealSrcAddress      string `json:"realSrcAddress"`
	SimulatedSrcAddress int    `json:"simulatedSrcAddress"`
	SimulatedDstAddress int    `json:"simulatedDstAddress"`
	MaxQueueLength      int    `json:"maxQueueLength"`
	MaxHops             int    `json:"maxHops"`
	DevName             string `json:"devName"`
	DevSrcAddr          string `json:"devSrcAddr"`
	DevDstAddr          string `json:"devDstAddr"`
	RoutingTableNum     string `json:"routingTableNum"`
	RoutingAlgorithm    string `json:"routingAlgorithm"`
}

type Config struct {
	Topology TopologyJson  `json:"topology"`
	General  GeneralConfig `json:"general"`
}
