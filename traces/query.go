package trace

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"

	config "github.com/aditiharini/simulator-proxy/config/experiment"
)

func ParseQuery(queryJson config.QueryJson) Query {
	if queryJson["type"] == "segment" {
		var segmentQuery SegmentQuery
		input := queryJson["input"].(config.QueryJson)
		segmentQuery.Input = ParseSingleOutputQuery(input)
		segmentQuery.NumSegments = int(queryJson["segments"].(float64))
		var outputs []string
		for _, output := range queryJson["output"].([]interface{}) {
			outputs = append(outputs, output.(string))
		}
		segmentQuery.Output = outputs
		return segmentQuery
	} else if queryJson["type"] == "range" {
		return ParseSingleOutputQuery(queryJson)
	} else if queryJson["type"] == "full_file" {
		return ParseSingleOutputQuery(queryJson)
	} else if queryJson["type"] == "stitch" {
		return ParseSingleOutputQuery(queryJson)
	} else if queryJson["type"] == "spotty" {
		return ParseSingleOutputQuery(queryJson)
	} else {
		panic("invalid query")
	}
}

func ParseSingleOutputQuery(queryJson config.QueryJson) SingleOutputQuery {
	jsonBytes, err := json.Marshal(queryJson)
	if err != nil {
		panic(err)
	}
	if queryJson["type"] == "range" {
		var rangeQuery RangeQuery
		input := queryJson["input"].(config.QueryJson)
		rangeQuery.Input = ParseSingleOutputQuery(input)
		rangeQuery.Length = int(queryJson["length"].(float64))
		rangeQuery.StartMilliOffset = int(queryJson["start"].(float64))
		rangeQuery.Output = queryJson["output"].(string)
		return rangeQuery
	} else if queryJson["type"] == "full_file" {
		var fullFileQuery FullFileQuery
		if err := json.Unmarshal(jsonBytes, &fullFileQuery); err != nil {
			panic(err)
		}
		return fullFileQuery
	} else if queryJson["type"] == "stitch" {
		var stitchQuery StitchQuery
		var queryInputs []Query
		inputs := queryJson["inputs"].([]config.QueryJson)
		for _, input := range inputs {
			queryInputs = append(queryInputs, ParseQuery(input))
		}
		stitchQuery.Inputs = queryInputs
		stitchQuery.Output = queryJson["output"].(string)
		return stitchQuery
	} else if queryJson["type"] == "spotty" {
		var spottyQuery SpottyQuery
		input := queryJson["input"].(config.QueryJson)
		spottyQuery.Input = ParseSingleOutputQuery(input)
		spottyQuery.Output = queryJson["output"].(string)
		spottyQuery.DisconnectThresholdLength = int(queryJson["disconnectThreshold"].(float64))
		spottyQuery.Length = int(queryJson["length"].(float64))
		spottyQuery.NumDisconnects = int(queryJson["disconnects"].(float64))
		return spottyQuery
	} else {
		panic("invalid query")
	}
}

func CreateScratchSpace() string {
	scratchDir := "scratch"
	if err := os.MkdirAll(scratchDir, os.ModePerm); err != nil {
		panic(err)
	}
	return scratchDir
}

func RemoveScratchSpace() {
	scratchDir := "scratch"
	if err := os.RemoveAll(scratchDir); err != nil {
		panic(err)
	}
}

type Query interface {
	Execute()
	Outfiles() []string
}

type SingleOutputQuery interface {
	Outfile() string
	Query
}

type RangeQuery struct {
	Input            SingleOutputQuery `json:"input"`
	StartMilliOffset int               `json:"start"`
	Length           int               `json:"length"`
	Output           string            `json:"output"`
}

func (rq RangeQuery) Execute() {
	rq.Input.Execute()
	rawTrace, err := os.Open(rq.Input.Outfile())
	if err != nil {
		panic(err)
	}
	rawTraceScanner := bufio.NewScanner(rawTrace)

	scratchDir := CreateScratchSpace()
	processedTraceTmp := fmt.Sprintf("%s/%s", scratchDir, rq.Output)
	processedTraceFile, err := os.Create(processedTraceTmp)
	if err != nil {
		panic(err)
	}

	processedTraceWriter := bufio.NewWriter(processedTraceFile)
	for rawTraceScanner.Scan() {
		offset, err := strconv.Atoi(rawTraceScanner.Text())
		if err != nil {
			panic(err)
		}
		if offset >= rq.StartMilliOffset && offset < rq.StartMilliOffset+rq.Length {
			newOffset := offset - rq.StartMilliOffset
			processedTraceWriter.WriteString(fmt.Sprintf("%d\n", newOffset))
		}
	}
	if err := rawTrace.Close(); err != nil {
		panic(err)
	}
	if err := processedTraceWriter.Flush(); err != nil {
		panic(err)
	}
	if err := processedTraceFile.Close(); err != nil {
		panic(err)
	}
	if err := os.Remove(rq.Input.Outfile()); err != nil {
		panic(err)
	}
	if err := os.Rename(processedTraceTmp, rq.Output); err != nil {
		panic(err)
	}
	RemoveScratchSpace()
}

func (rq RangeQuery) Outfile() string {
	return rq.Output
}

func (rq RangeQuery) Outfiles() []string {
	return []string{rq.Output}
}

type SegmentQuery struct {
	Input       SingleOutputQuery `json:"input"`
	NumSegments int               `json:"segments"`
	Output      []string          `json:"output"`
}

func (sq SegmentQuery) Execute() {
	sq.Input.Execute()
	rawTrace, err := os.Open(sq.Input.Outfile())
	if err != nil {
		panic(err)
	}
	defer rawTrace.Close()
	rawTraceScanner := bufio.NewScanner(rawTrace)
	lastTime := ""
	for rawTraceScanner.Scan() {
		lastTime = rawTraceScanner.Text()
	}
	duration, err := strconv.Atoi(lastTime)
	if err != nil {
		panic(err)
	}
	_, err = rawTrace.Seek(0, io.SeekStart)
	if err != nil {
		panic(err)
	}

	durationPerSegment := duration / sq.NumSegments
	rawTraceScanner = bufio.NewScanner(rawTrace)
	scratchDir := CreateScratchSpace()
	if err := os.Chdir(scratchDir); err != nil {
		panic(err)
	}
	processedTraceFile, err := os.Create(sq.Output[0])
	if err != nil {
		panic(err)
	}
	processedTraceWriter := bufio.NewWriter(processedTraceFile)
	nextSegmentNumber := 1
	for rawTraceScanner.Scan() {
		offset, err := strconv.Atoi(rawTraceScanner.Text())
		if err != nil {
			panic(err)
		}
		if nextSegmentNumber < sq.NumSegments && offset > nextSegmentNumber*durationPerSegment {
			processedTraceWriter.Flush()
			processedTraceFile.Close()
			processedTraceFile, err = os.Create(sq.Output[nextSegmentNumber])
			if err != nil {
				panic(err)
			}
			processedTraceWriter = bufio.NewWriter(processedTraceFile)
			nextSegmentNumber++
		}
		newOffset := offset - ((nextSegmentNumber - 1) * durationPerSegment)
		processedTraceWriter.WriteString(fmt.Sprintf("%d\n", newOffset))
	}
	processedTraceWriter.Flush()
	processedTraceFile.Close()
	if err := os.Chdir(".."); err != nil {
		panic(err)
	}
	if err := os.Remove(sq.Input.Outfile()); err != nil {
		panic(err)
	}
	for _, output := range sq.Output {
		if err := os.Rename(fmt.Sprintf("%s/%s", scratchDir, output), output); err != nil {
			panic(err)
		}
	}
	RemoveScratchSpace()
}

func (sq SegmentQuery) Outfiles() []string {
	return sq.Output
}

type FullFileQuery struct {
	Batchname string `json:"batch"`
	Tracename string `json:"trace"`
	Output    string `json:"output`
}

func (fq FullFileQuery) Execute() {
	if err := exec.Command("dropbox_uploader.sh", "download", GetRemoteTracePath(fq.Batchname, fq.Tracename), fq.Output).Run(); err != nil {
		panic(err)
	}
}

func (fq FullFileQuery) Outfile() string {
	return fq.Output
}

func (fq FullFileQuery) Outfiles() []string {
	return []string{fq.Output}
}

type StitchQuery struct {
	Inputs []Query `json:"inputs"`
	Output string  `json:"output"`
}

func (sq StitchQuery) Execute() {
	var allInputs []string
	for _, input := range sq.Inputs {
		input.Execute()
		allInputs = append(allInputs, input.Outfiles()...)
	}

	scratchDir := CreateScratchSpace()
	processedTraceFile, err := os.Create(fmt.Sprintf("%s/%s", scratchDir, sq.Output))
	if err != nil {
		panic(err)
	}
	defer processedTraceFile.Close()
	processedTraceWriter := bufio.NewWriter(processedTraceFile)
	defer processedTraceWriter.Flush()

	lastOffset := 0
	for _, input := range allInputs {
		rawTraceFile, err := os.Open(input)
		if err != nil {
			panic(err)
		}
		defer rawTraceFile.Close()
		rawTraceScanner := bufio.NewScanner(rawTraceFile)
		curOffset := 0
		for rawTraceScanner.Scan() {
			offset, err := strconv.Atoi(rawTraceScanner.Text())
			if err != nil {
				panic(err)
			}
			newOffset := lastOffset + offset
			curOffset = newOffset
			processedTraceWriter.WriteString(fmt.Sprintf("%d\n", newOffset))
		}
		lastOffset = curOffset
	}

	for _, input := range allInputs {
		if err := os.Remove(input); err != nil {
			panic(err)
		}
	}
	if err := os.Rename(fmt.Sprintf("%s/%s", scratchDir, sq.Output), sq.Output); err != nil {
		panic(err)
	}
	RemoveScratchSpace()
}

func (sq StitchQuery) Outfile() string {
	return sq.Output
}

func (sq StitchQuery) Outfiles() []string {
	return []string{sq.Output}
}

type SpottyQuery struct {
	Input                     SingleOutputQuery
	Output                    string
	DisconnectThresholdLength int
	Length                    int
	NumDisconnects            int
}

func (sq SpottyQuery) Execute() {
	sq.Input.Execute()
	infile := sq.Input.Outfile()
	rawTraceFileMin, err := os.Open(infile)
	if err != nil {
		panic(err)
	}
	defer rawTraceFileMin.Close()
	rawTraceFileMax, err := os.Open(infile)
	if err != nil {
		panic(err)
	}
	defer rawTraceFileMax.Close()
	rawTraceReaderMin := bufio.NewScanner(rawTraceFileMin)
	rawTraceReaderMax := bufio.NewScanner(rawTraceFileMax)
	minOffset, maxOffset := -1, -1
	var disconnects []int
	prevOffset := 0
	foundSection := false
	for rawTraceReaderMax.Scan() {
		maxOffset, err = strconv.Atoi(rawTraceReaderMax.Text())
		if err != nil {
			panic(err)
		}
		if minOffset == -1 {
			minOffset = maxOffset
		}

		if prevOffset != 0 && maxOffset >= prevOffset+sq.DisconnectThresholdLength {
			disconnects = append(disconnects, maxOffset)
		}

		for maxOffset-minOffset > sq.Length {
			if len(disconnects) >= sq.NumDisconnects {
				foundSection = true
				break
			}
			if !rawTraceReaderMin.Scan() {
				panic(fmt.Sprintf("min ahead of max (min: %d, max:%d)", minOffset, maxOffset))
			}
			minOffset, err = strconv.Atoi(rawTraceReaderMin.Text())
			if len(disconnects) > 0 && minOffset > disconnects[0] {
				disconnects = disconnects[1:]
			}
		}

		if foundSection {
			break
		}
		prevOffset = maxOffset
	}

	if foundSection {
		fmt.Printf("Range: (%d, %d), Disconnects:%d\n", minOffset, maxOffset, len(disconnects))
		rangeQuery := RangeQuery{Input: sq.Input, Output: sq.Output, StartMilliOffset: minOffset, Length: maxOffset - minOffset}
		rangeQuery.Execute()
	} else {
		panic("unsatisfiable query")
	}
}

func (sq SpottyQuery) Outfile() string {
	return sq.Output
}

func (sq SpottyQuery) Outfiles() []string {
	return []string{sq.Output}
}

func GetRemoteTracePath(batchName string, traceName string) string {
	return fmt.Sprintf("Drone-Project/measurements/iperf_traces/%s/processed/traces/%s", batchName, traceName)
}
