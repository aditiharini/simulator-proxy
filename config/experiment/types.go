package config

import (
	"fmt"

	receiverConfig "github.com/aditiharini/simulator-proxy/config/packet-receiver"
	senderConfig "github.com/aditiharini/simulator-proxy/config/packet-sender"
	simulatorConfig "github.com/aditiharini/simulator-proxy/config/simulator"
)

type FullyConnectedJson = map[string]string

type DroneLinkConfig struct {
	Type  string `json:"type"`
	Delay int    `json:"delay"`
}

type SimulatorConfig struct {
	Timeout    int                           `json:"timeout"`
	DroneLinks DroneLinkConfig               `json:"droneLinks"`
	BaseLinks  FullyConnectedJson            `json:"baseLinks"`
	Global     simulatorConfig.GeneralConfig `json:"global"`
}

type QueryJson = map[string]interface{}

type Config struct {
	Sender     senderConfig.Config   `json:"sender"`
	Receiver   receiverConfig.Config `json:"receiver"`
	Simulator  SimulatorConfig       `json:"simulator"`
	Query      []QueryJson           `json:"query"`
	Evaluation Evaluation            `json:"evaluation"`
}

type EvaluationSetup struct {
	Input   string   `json:"input"`
	Args    string   `json:"args"`
	Script  string   `json:"script"`
	Outputs []string `json:"outputs"`
}

type Evaluation struct {
	Dir    string            `json:"dir"`
	Setups []EvaluationSetup `json:"setup"`
}

// TODO(aditi): Having an empty interface like this is bad
type ConfigEntry interface {
}

type DelayEntry struct {
	EntryType   string `json:"type"`
	DelayMillis int    `json:"delay"`
}

func NewDelayEntry(delayMillis int) DelayEntry {
	return DelayEntry{EntryType: "delay", DelayMillis: delayMillis}
}

type TraceEntry struct {
	EntryType string `json:"type"`
	TraceFile string `json:"file"`
	LossFile  string `json:"loss"`
}

func NewTraceEntry(tracefile string) TraceEntry {
	return TraceEntry{EntryType: "trace", TraceFile: fmt.Sprintf("%s.pps", tracefile), LossFile: fmt.Sprintf("%s.loss", tracefile)}
}
