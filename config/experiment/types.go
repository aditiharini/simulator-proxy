package config

import (
	receiverConfig "github.com/aditiharini/simulator-proxy/config/packet-receiver"
	senderConfig "github.com/aditiharini/simulator-proxy/config/packet-sender"
	simulatorConfig "github.com/aditiharini/simulator-proxy/config/simulator"
)

type FullyConnectedJson = map[string]string

type SimulatorConfig struct {
	Topology FullyConnectedJson            `json:"topology"`
	Global   simulatorConfig.GeneralConfig `json:"global"`
}

type Config struct {
	Sender    senderConfig.Config   `json:"sender"`
	Receiver  receiverConfig.Config `json:"receiver"`
	Simulator SimulatorConfig       `json:"simulator"`
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
}

func NewTraceEntry(tracefile string) TraceEntry {
	return TraceEntry{EntryType: "trace", TraceFile: tracefile}
}
