package trace

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
)

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

func RemoveInputs(input string) {
	if err := os.Remove(fmt.Sprintf("%s.pps", input)); err != nil {
		panic(err)
	}
	if err := os.Remove(fmt.Sprintf("%s.loss", input)); err != nil {
		panic(err)
	}
}

func RenameOutputs(output string, scratchDir string) {
	if err := os.Rename(fmt.Sprintf("%s/%s.pps", scratchDir, output), fmt.Sprintf("%s.pps", output)); err != nil {
		panic(err)
	}
	if err := os.Rename(fmt.Sprintf("%s/%s.loss", scratchDir, output), fmt.Sprintf("%s.loss", output)); err != nil {
		panic(err)
	}
}

func RunProcessing(inputs []string, outputs []string, doProcessing func([]*bufio.Scanner, []*csv.Reader, []*bufio.Writer, []*csv.Writer)) {
	var rawTraceScanners []*bufio.Scanner
	var rawLossTraceReaders []*csv.Reader
	for _, input := range inputs {
		rawTrace, err := os.Open(fmt.Sprintf("%s.pps", input))
		if err != nil {
			panic(err)
		}
		defer rawTrace.Close()
		rawTraceScanner := bufio.NewScanner(rawTrace)

		rawLossTrace, err := os.Open(fmt.Sprintf("%s.loss", input))
		if err != nil {
			panic(err)
		}
		defer rawLossTrace.Close()
		rawLossTraceReader := csv.NewReader(rawLossTrace)

		rawTraceScanners = append(rawTraceScanners, rawTraceScanner)
		rawLossTraceReaders = append(rawLossTraceReaders, rawLossTraceReader)
	}

	scratchDir := CreateScratchSpace()
	if err := os.Chdir(scratchDir); err != nil {
		panic(err)
	}

	var traceWriters []*bufio.Writer
	var lossWriters []*csv.Writer
	for _, output := range outputs {
		processedTraceFile, err := os.Create(fmt.Sprintf("%s.pps", output))
		defer processedTraceFile.Close()
		if err != nil {
			panic(err)
		}

		traceWriter := bufio.NewWriter(processedTraceFile)
		defer traceWriter.Flush()
		traceWriters = append(traceWriters, traceWriter)

		processedLossFile, err := os.Create(fmt.Sprintf("%s.loss", output))
		lossWriters = append(lossWriters, csv.NewWriter(processedLossFile))
	}

	doProcessing(rawTraceScanners, rawLossTraceReaders, traceWriters, lossWriters)

	if err := os.Chdir(".."); err != nil {
		panic(err)
	}

	for _, input := range inputs {
		RemoveInputs(input)
	}
	for _, output := range outputs {
		RenameOutputs(output, scratchDir)
	}
}

func GetRemoteTracePath(batchName string, traceName string) string {
	return fmt.Sprintf("Drone-Project/measurements/iperf_traces/%s/processed/traces/%s", batchName, traceName)
}

func ForEachOffsetFile(tracefileName string, operator func(offset int)) {
	tracefile, err := os.Open(tracefileName)
	if err != nil {
		panic(err)
	}
	defer tracefile.Close()
	tracefileScanner := bufio.NewScanner(tracefile)

	ForEachOffsetScanner(tracefileScanner, operator)
}

func ForEachOffsetScanner(scanner *bufio.Scanner, operator func(offset int)) {
	for scanner.Scan() {
		offset, err := strconv.Atoi(scanner.Text())
		if err != nil {
			panic(err)
		}

		operator(offset)
	}
}

func ForEachLossReader(reader *csv.Reader, operator func(offset int, probability string)) {
	for {
		line, err := reader.Read()
		if err != nil {
			panic(err)
		}

		offset, err := strconv.Atoi(line[0])
		if err != nil {
			panic(err)
		}

		operator(offset, line[1])
	}
}
