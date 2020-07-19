package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	. "github.com/aditiharini/simulator-proxy/simulation"
)

type simTime struct {
	time.Time
}

func (t *simTime) UnmarshalJSON(buf []byte) error {
	tt, err := time.Parse(time.StampMicro, strings.Trim(string(buf), `"`))
	if err != nil {
		return err
	}
	t.Time = tt
	return nil
}

type Stats struct {
	entryTime     map[int]simTime
	firstExitTime map[int]simTime
}

func (s Stats) prettyPrint() {
	latencies := s.calculateLatencies()
	fmt.Println(latencies)
}

func (s Stats) calculateLatencies() map[int]time.Duration {
	latencyMap := make(map[int]time.Duration)
	for id, entry := range s.entryTime {
		exit := s.firstExitTime[id]
		latencyMap[id] = exit.Sub(entry.Time)
	}
	return latencyMap
}

type Event interface {
	process(stats *Stats)
}

type PacketSentEvent struct {
	Id   int     `json:"id"`
	Src  Address `json:"src"`
	Time simTime `json:"time"`
}

func (e PacketSentEvent) process(stats *Stats) {
	if _, ok := stats.firstExitTime[e.Id]; !ok {
		stats.firstExitTime[e.Id] = e.Time
	}
}

type PacketReceivedEvent struct {
	Id   int     `json:"id"`
	Time simTime `json:"time"`
}

func (e PacketReceivedEvent) process(stats *Stats) {
	stats.entryTime[e.Id] = e.Time
}

type StartTraceEvent struct {
	Src  Address `json:"src"`
	Dst  Address `json:"dst"`
	Time simTime `json:"time"`
}

func (e StartTraceEvent) process(stats *Stats) {
}

func parseLogLine(data []byte) Event {
	var mappedData map[string]interface{}
	json.Unmarshal(data, &mappedData)
	if mappedData["event"] == "packet_received" {
		var packetReceived PacketReceivedEvent
		json.Unmarshal(data, &packetReceived)
		return packetReceived
	} else if mappedData["event"] == "packet_sent" {
		var packetSent PacketSentEvent
		json.Unmarshal(data, &packetSent)
		return packetSent
	} else if mappedData["event"] == "start_trace" {
		var startTrace StartTraceEvent
		json.Unmarshal(data, &startTrace)
		return startTrace
	} else {
		panic("unrecognized event type")
	}
}

func main() {
	filename := os.Args[1]

	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	stats := Stats{
		entryTime:     make(map[int]simTime),
		firstExitTime: make(map[int]simTime),
	}
	for scanner.Scan() {
		event := parseLogLine(scanner.Bytes())
		event.process(&stats)
	}
	stats.prettyPrint()

}
