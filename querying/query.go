package querying

import (
	"bufio"
	"encoding/csv"
	"fmt"
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
	Input            SingleOutputQuery `json:"input"`
	StartMilliOffset int               `json:"start"`
	Length           int               `json:"length"`
	Output           string            `json:"output"`
}

func (rq RangeQuery) Execute() {
	rq.Input.Execute()
	rq.ExecuteWithInputFile()
}

func (rq RangeQuery) ExecuteWithInputFile() {
	RunProcessing([]string{rq.Input.Outfile()}, []string{rq.Output}, func(traceReaders []*bufio.Scanner, lossReaders []*csv.Reader, traceWriters []*bufio.Writer, lossWriters []*csv.Writer) {
		lastWrittenOffset := 0

		rawTraceScanner, rawLossTraceReader := traceReaders[0], lossReaders[0]
		processedTraceWriter, processedLossWriter := traceWriters[0], lossWriters[0]

		ForEachOffsetScanner(rawTraceScanner, func(offset int) {
			if offset >= rq.StartMilliOffset && offset < rq.StartMilliOffset+rq.Length {
				newOffset := offset - rq.StartMilliOffset
				processedTraceWriter.WriteString(fmt.Sprintf("%d\n", newOffset))
				lastWrittenOffset = newOffset
			}
		})

		ForEachLossReader(rawLossTraceReader, func(offset int, probability string) {
			if offset >= rq.StartMilliOffset && offset < rq.StartMilliOffset+rq.Length {
				newOffset := offset - rq.StartMilliOffset
				processedLossWriter.Write([]string{fmt.Sprintf("%d", newOffset), probability})
			}
		})

		processedLossWriter.Write([]string{fmt.Sprintf("%d", lastWrittenOffset), fmt.Sprintf("%f", 0.)})
	})
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

	duration := 0
	ForEachOffsetFile(fmt.Sprintf("%s.pps", sq.Input.Outfile()), func(offset int) {
		duration = offset
	})

	durationPerSegment := duration / sq.NumSegments
	RunProcessing([]string{sq.Input.Outfile()}, sq.Output, func(traceReaders []*bufio.Scanner, lossReaders []*csv.Reader, traceWriters []*bufio.Writer, lossWriters []*csv.Writer) {
		rawTraceScanner, lossTraceReader := traceReaders[0], lossReaders[0]
		processedTraceWriter := traceWriters[0]
		nextSegmentNumber := 1
		segmentStarts := []int{0}
		ForEachOffsetScanner(rawTraceScanner, func(offset int) {
			if nextSegmentNumber < sq.NumSegments && offset > nextSegmentNumber*durationPerSegment {
				processedTraceWriter = traceWriters[nextSegmentNumber]
				nextSegmentNumber++
				segmentStarts = append(segmentStarts, offset)
			}
			newOffset := offset - ((nextSegmentNumber - 1) * durationPerSegment)
			processedTraceWriter.WriteString(fmt.Sprintf("%d\n", newOffset))
		})

		lossWriter := lossWriters[0]
		nextSegmentNumber = 1
		ForEachLossReader(lossTraceReader, func(offset int, probability string) {
			if nextSegmentNumber < len(segmentStarts) && offset >= segmentStarts[nextSegmentNumber] {
				newOffset := offset - ((nextSegmentNumber - 1) * durationPerSegment)
				lossWriter.Write([]string{fmt.Sprintf("%d", newOffset), fmt.Sprintf("%f", 0.)})

				lossWriter = lossWriters[nextSegmentNumber]
				nextSegmentNumber++
			}
			newOffset := offset - ((nextSegmentNumber - 1) * durationPerSegment)
			lossWriter.Write([]string{fmt.Sprintf("%d", newOffset), probability})

		})
	})
}

func (sq SegmentQuery) Outfiles() []string {
	return sq.Output
}

type FullFileQuery struct {
	Batchname string `json:"batch"`
	Tracename string `json:"trace"`
	Output    string `json:"output"`
}

func (fq FullFileQuery) Execute() {
	if out, err := exec.Command("dropbox_uploader.sh", "download", fmt.Sprintf("%s.pps", GetRemoteTracePath(fq.Batchname, fq.Tracename)), fmt.Sprintf("%s.pps", fq.Output)).CombinedOutput(); err != nil {
		print(string(out))
		panic(err)
	}
	if out, err := exec.Command("dropbox_uploader.sh", "download", fmt.Sprintf("%s.loss", GetRemoteTracePath(fq.Batchname, fq.Tracename)), fmt.Sprintf("%s.loss", fq.Output)).CombinedOutput(); err != nil {
		print(string(out))
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

	RunProcessing(allInputs, []string{sq.Output}, func(traceReaders []*bufio.Scanner, lossReaders []*csv.Reader, traceWriters []*bufio.Writer, lossWriters []*csv.Writer) {
		lastOffset := 0
		processedTraceWriter := traceWriters[0]
		for _, traceReader := range traceReaders {
			curOffset := 0
			ForEachOffsetScanner(traceReader, func(offset int) {
				newOffset := lastOffset + offset
				curOffset = newOffset
				processedTraceWriter.WriteString(fmt.Sprintf("%d\n", newOffset))
			})
			lastOffset = curOffset
		}

		lastOffset = 0
		processedLossWriter := lossWriters[0]
		for _, lossReader := range lossReaders {
			curOffset := 0
			ForEachLossReader(lossReader, func(offset int, probability string) {
				newOffset := lastOffset + offset
				curOffset = newOffset
				processedLossWriter.Write([]string{fmt.Sprintf("%d", newOffset), probability})
			})
			lastOffset = curOffset
		}
	})

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

	RunProcessing([]string{sq.Input.Outfile(), sq.Input.Outfile()}, []string{sq.Output}, func(traceReaders []*bufio.Scanner, lossReaders []*csv.Reader, traceWriters []*bufio.Writer, lossWriters []*csv.Writer) {
		rawTraceReaderMin, rawTraceReaderMax := traceReaders[0], traceReaders[1]
		minOffsetOverall, maxOffsetOverall := -1, -1
		var disconnects []int
		prevOffset := 0
		foundSection := false
		ForEachOffsetScanner(rawTraceReaderMax, func(maxOffset int) {
			if !foundSection {
				maxOffsetOverall = maxOffset
				if minOffsetOverall == -1 {
					minOffsetOverall = maxOffset
				}

				if prevOffset != 0 && maxOffset >= prevOffset+sq.DisconnectThresholdLength {
					disconnects = append(disconnects, maxOffset)
				}

				for maxOffset-minOffsetOverall > sq.Length {
					if len(disconnects) >= sq.NumDisconnects {
						foundSection = true
						break
					}
					if !rawTraceReaderMin.Scan() {
						panic(fmt.Sprintf("min ahead of max (min: %d, max:%d), error: %v", minOffsetOverall, maxOffset, rawTraceReaderMin.Err()))
					}
					minOffsetInt, err := strconv.Atoi(rawTraceReaderMin.Text())
					minOffsetOverall = minOffsetInt

					if err != nil {
						panic(err)
					}
					if len(disconnects) > 0 && minOffsetOverall > disconnects[0] {
						disconnects = disconnects[1:]
					}
				}
				prevOffset = maxOffset
			}
		})
		if foundSection {
			fmt.Printf("Range: (%d, %d), Disconnects:%d\n", minOffsetOverall, maxOffsetOverall, len(disconnects))

			CopyFile(fmt.Sprintf("%s.pps", sq.Input.Outfile()), fmt.Sprintf("../%s.pps", sq.Input.Outfile()))
			CopyFile(fmt.Sprintf("%s.loss", sq.Input.Outfile()), fmt.Sprintf("../%s.loss", sq.Input.Outfile()))

			rangeQuery := RangeQuery{Input: sq.Input, Output: sq.Output, StartMilliOffset: minOffsetOverall, Length: maxOffsetOverall - minOffsetOverall}
			rangeQuery.ExecuteWithInputFile()
		} else {
			panic("unsatisfiable query")
		}
	})
}

func (sq SpottyQuery) Outfile() string {
	return sq.Output
}

func (sq SpottyQuery) Outfiles() []string {
	return []string{sq.Output}
}
