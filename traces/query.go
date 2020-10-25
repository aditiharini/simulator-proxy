package trace

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
)

type Query interface {
	Execute()
	Outfiles() []string
}

type SingleOutputQuery interface {
	Outfile() string
	Query
}

type RangeQuery struct {
	input            SingleOutputQuery
	startMilliOffset int
	length           int
	outfile          string
}

func (rq RangeQuery) Execute() {
	rq.input.Execute()
	rawTrace, err := os.Open(rq.input.Outfile())
	if err != nil {
		panic(err)
	}
	defer rawTrace.Close()
	rawTraceScanner := bufio.NewScanner(rawTrace)
	processedTraceFile, err := os.Create(rq.outfile)
	if err != nil {
		panic(err)
	}
	defer processedTraceFile.Close()
	processedTraceWriter := bufio.NewWriter(processedTraceFile)
	defer processedTraceWriter.Flush()
	for rawTraceScanner.Scan() {
		offset, err := strconv.Atoi(rawTraceScanner.Text())
		if err != nil {
			panic(err)
		}
		if offset >= rq.startMilliOffset && offset < rq.startMilliOffset+rq.length {
			newOffset := offset - rq.startMilliOffset
			processedTraceWriter.WriteString(fmt.Sprintf("%d\n", newOffset))
		}
	}
}

func (rq RangeQuery) Outfile() string {
	return rq.outfile
}

func (rq RangeQuery) Outfiles() []string {
	return []string{rq.outfile}
}

type SegmentQuery struct {
	input       SingleOutputQuery
	numSegments int
	outfiles    []string
}

func (sq SegmentQuery) Execute() {
	sq.input.Execute()
	rawTrace, err := os.Open(sq.input.Outfile())
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

	durationPerSegment := duration / sq.numSegments
	rawTraceScanner = bufio.NewScanner(rawTrace)
	processedTraceFile, err := os.Create(sq.outfiles[0])
	if err != nil {
		panic(err)
	}
	defer processedTraceFile.Close()
	processedTraceWriter := bufio.NewWriter(processedTraceFile)
	defer processedTraceWriter.Flush()
	currentSegmentNumber := 0
	for rawTraceScanner.Scan() {
		offset, err := strconv.Atoi(rawTraceScanner.Text())
		if err != nil {
			panic(err)
		}
		if offset > (currentSegmentNumber+1)*durationPerSegment {
			processedTraceFile, err = os.Create(sq.outfiles[currentSegmentNumber])
			if err != nil {
				panic(err)
			}
			defer processedTraceFile.Close()
			processedTraceWriter = bufio.NewWriter(processedTraceFile)
			defer processedTraceWriter.Flush()
			currentSegmentNumber++
		}
		newOffset := offset - (currentSegmentNumber * durationPerSegment)
		processedTraceWriter.WriteString(fmt.Sprintf("%d\n", newOffset))
	}
}

func (sq SegmentQuery) Outfiles() []string {
	return sq.outfiles
}

type FullFileQuery struct {
	batchname string
	outfile   string
}

func (fq FullFileQuery) Execute() {
	if err := exec.Command("dropbox_uploader.sh", "download", GetRemoteTracePath(fq.batchname), fq.outfile); err != nil {
		panic(err)
	}
}

func (fq FullFileQuery) Outfile() string {
	return fq.outfile
}

func (fq FullFileQuery) Outfiles() []string {
	return []string{fq.outfile}
}

type StitchQuery struct {
	inputs  []Query
	outfile string
}

func (sq StitchQuery) Execute() {
	var allInputs []string
	for _, input := range sq.inputs {
		input.Execute()
		allInputs = append(allInputs, input.Outfiles()...)
	}

	processedTraceFile, err := os.Create(sq.outfile)
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
}

func (sq StitchQuery) Outfile() string {
	return sq.outfile
}

func (sq StitchQuery) Outfiles() []string {
	return []string{sq.outfile}
}

func GetRemoteTracePath(batchName string) string {
	return fmt.Sprintf("~/Drone-Project/measurements/iperf_traces/%s/processed/traces/", batchName)
}
