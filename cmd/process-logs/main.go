package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	. "github.com/aditiharini/simulator-proxy/simulation"
)

type PacketId = int

type simTime struct {
	time.Time
}
type OffsetTime struct {
	offset time.Duration
	base   simTime
}

func (t *OffsetTime) normalizeTo(newBase simTime) {
	oldBase := t.base
	difference := oldBase.Sub(newBase.Time)
	t.base = newBase
	t.offset = t.offset + difference
}

func (t *OffsetTime) replaceBaseWithoutNormalizing(newBase simTime) {
	t.base = newBase
}

func (t *simTime) UnmarshalJSON(buf []byte) error {
	tt, err := time.Parse(time.StampMicro, strings.Trim(string(buf), `"`))
	if err != nil {
		return err
	}
	t.Time = tt
	return nil
}

type Graph interface {
	addDataset(data []interface{})
	toCsv()
}

type LatencyData struct {
	time    OffsetTime
	latency time.Duration
	dropped bool
}

type Dataset interface {
	getColumnNames() []string
}

type LatencyDataset struct {
	data []LatencyData
}

func (ld *LatencyDataset) getColumnNames() []string {
	return []string{"time", "latency"}
}

func (ld *LatencyDataset) replaceBaseTimes(newBase simTime) {
	for _, data := range ld.data {
		data.time.replaceBaseWithoutNormalizing(newBase)
	}
}

func (ld *LatencyDataset) normalizeTimesTo(newBase simTime) {
	for _, data := range ld.data {
		data.time.normalizeTo(newBase)
	}
}

func (ld LatencyData) toStringList() []string {
	time := fmt.Sprintf("%d", ld.time.offset.Milliseconds())
	latency := fmt.Sprintf("%d", ld.latency.Milliseconds())
	dropped := fmt.Sprintf("%v", ld.dropped)
	return []string{time, latency, dropped}
}

func experimentType(filename string) string {
	parts := strings.Split(filename, "/")
	name := strings.Split(parts[len(parts)-1], ".csv")[0]
	return name
}

func (lg *LatencyDataset) toCsv(filename string) {
	file, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	w := csv.NewWriter(file)
	defer w.Flush()

	experimentType := experimentType(filename)
	columns := []string{"time", "latency", "dropped", "type"}
	w.Write(columns)
	for _, latencyData := range lg.data {
		w.Write(append(latencyData.toStringList(), experimentType))
	}
}

type Link struct {
	src int
	dst int
}

type Stats struct {
	entryTime        map[PacketId]simTime
	firstExitTime    map[PacketId]simTime
	perLinkEntryTime map[Link](map[PacketId]simTime)
	perLinkExitTime  map[Link](map[PacketId]simTime)
	startTime        simTime
	perLinkStartTime map[Link]simTime
}

func (s Stats) getTimeAsOffsetFromGlobalStart(eventTime simTime) OffsetTime {
	baseTime := s.startTime
	return OffsetTime{offset: eventTime.Sub(baseTime.Time), base: baseTime}
}

func (s Stats) getTimeAsOffsetFromLinkStart(eventTime simTime, link Link) OffsetTime {
	baseTime := s.perLinkStartTime[link]
	return OffsetTime{offset: eventTime.Sub(baseTime.Time), base: baseTime}
}

func (s Stats) calculateLatencies() []LatencyData {
	var latencyData []LatencyData
	for id, entry := range s.entryTime {
		offsetTime := s.getTimeAsOffsetFromGlobalStart(entry)
		if exit, ok := s.firstExitTime[id]; ok {
			latencyData = append(latencyData, LatencyData{time: offsetTime, latency: exit.Sub(entry.Time), dropped: false})
		} else {
			latencyData = append(latencyData, LatencyData{time: offsetTime, dropped: true})
		}
	}
	return latencyData
}

func (s Stats) calculatePerLinkLatencies(link Link) []LatencyData {
	var latencyData []LatencyData
	for id, entryTime := range s.perLinkEntryTime[link] {
		offsetTime := s.getTimeAsOffsetFromLinkStart(entryTime, link)
		if exit, ok := s.perLinkExitTime[link][id]; ok {
			latencyData = append(latencyData, LatencyData{time: offsetTime, latency: exit.Sub(entryTime.Time), dropped: false})
		} else {
			latencyData = append(latencyData, LatencyData{time: offsetTime, dropped: true})
		}
	}
	return latencyData
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
	stats.perLinkStartTime[Link{src: e.Src, dst: e.Dst}] = e.Time
}

type StartSimulatorEvent struct {
	Time simTime `json:"time"`
}

func (e StartSimulatorEvent) process(stats *Stats) {
	stats.startTime = e.Time
}

type PacketEnteredLinkEvent struct {
	Id   int     `json:"id"`
	Src  Address `json:"src"`
	Dst  Address `json:"dst"`
	Time simTime `json:"time"`
}

func (e PacketEnteredLinkEvent) process(stats *Stats) {
	link := Link{src: e.Src, dst: e.Dst}
	if _, ok := stats.perLinkEntryTime[link]; !ok {
		stats.perLinkEntryTime[link] = make(map[int]simTime)
	}
	stats.perLinkEntryTime[link][e.Id] = e.Time
}

type PacketLeftLinkEvent struct {
	Id   int     `json:"id"`
	Src  Address `json:"src"`
	Dst  Address `json:"dst"`
	Time simTime `json:"time"`
}

func (e PacketLeftLinkEvent) process(stats *Stats) {
	link := Link{src: e.Src, dst: e.Dst}
	if _, ok := stats.perLinkExitTime[link]; !ok {
		stats.perLinkExitTime[link] = make(map[int]simTime)
	}
	stats.perLinkExitTime[Link{src: e.Src, dst: e.Dst}][e.Id] = e.Time
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
	} else if mappedData["event"] == "packet_left_link" {
		var packetLeftLink PacketLeftLinkEvent
		json.Unmarshal(data, &packetLeftLink)
		return packetLeftLink
	} else if mappedData["event"] == "packet_entered_link" {
		var packetEnteredLink PacketEnteredLinkEvent
		json.Unmarshal(data, &packetEnteredLink)
		return packetEnteredLink
	} else if mappedData["event"] == "start_simulator" {
		var startSimulator StartSimulatorEvent
		json.Unmarshal(data, &startSimulator)
		return startSimulator
	} else {
		panic(fmt.Sprintf("unrecognized event type in message:%v, original: %s", mappedData, string(data)))
	}
}

func splitLinkLogs(combined string) []string {
	return strings.Split(combined, ",")
}

func combineCsvs(csvs []string, outfname string) {
	outFile, err := os.Create(outfname)
	if err != nil {
		panic(err)
	}
	defer outFile.Close()
	outWriter := csv.NewWriter(outFile)
	outWriter.Write([]string{"time", "latency", "dropped", "type"})
	defer outWriter.Flush()
	for _, fname := range csvs {
		csvFile, err := os.Open(fname)
		if err != nil {
			panic(err)
		}

		reader := csv.NewReader(csvFile)
		// Discard the header
		reader.Read()
		for {
			row, err := reader.Read()
			if err == io.EOF {
				break
			} else if err != nil {
				panic(err)
			}
			if err := outWriter.Write(row); err != nil {
				panic(err)
			}
		}
		csvFile.Close()
	}
}

func main() {
	baseStation := 999
	newLog := flag.String("newlog", "full.log", "file name of experiment log")
	linkLogs := flag.String("linkLogs", "1.log,1.log,1.log", "file name of single link logs")
	outdir := flag.String("outdir", "tmp", "where to write output link csvs")
	flag.Parse()

	file, err := os.Open(*newLog)
	if err != nil {
		panic(err)
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	stats := Stats{
		entryTime:        make(map[PacketId]simTime),
		firstExitTime:    make(map[PacketId]simTime),
		perLinkEntryTime: make(map[Link](map[PacketId]simTime)),
		perLinkExitTime:  make(map[Link](map[PacketId]simTime)),
		perLinkStartTime: make(map[Link]simTime),
	}

	for scanner.Scan() {
		event := parseLogLine(scanner.Bytes())
		event.process(&stats)
	}

	var allCsvs []string
	combinedDataset := LatencyDataset{data: stats.calculateLatencies()}
	combinedPath := fmt.Sprintf("%s/combined.csv", *outdir)
	combinedDataset.toCsv(combinedPath)
	allCsvs = append(allCsvs, combinedPath)

	for i, linkLog := range splitLinkLogs(*linkLogs) {
		file, err := os.Open(linkLog)
		if err != nil {
			panic(err)
		}

		defer file.Close()

		scanner := bufio.NewScanner(file)

		linkStats := Stats{
			entryTime:        make(map[PacketId]simTime),
			firstExitTime:    make(map[PacketId]simTime),
			perLinkEntryTime: make(map[Link](map[PacketId]simTime)),
			perLinkExitTime:  make(map[Link](map[PacketId]simTime)),
			perLinkStartTime: make(map[Link]simTime),
		}

		for scanner.Scan() {
			event := parseLogLine(scanner.Bytes())
			event.process(&linkStats)
		}

		latencies := linkStats.calculatePerLinkLatencies(Link{src: 0, dst: baseStation})
		correspondingLinkStart := stats.perLinkStartTime[Link{src: i, dst: baseStation}]
		linkDataset := LatencyDataset{data: latencies}
		linkDataset.replaceBaseTimes(correspondingLinkStart)
		linkDataset.normalizeTimesTo(stats.startTime)
		linkPath := fmt.Sprintf("%s/link%d.csv", *outdir, i)
		allCsvs = append(allCsvs, linkPath)
		linkDataset.toCsv(linkPath)
	}

	combineCsvs(allCsvs, fmt.Sprintf("%s/all.csv", *outdir))
}
