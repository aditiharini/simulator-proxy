package simulation

import (
	"encoding/csv"
	"io"
	"math/rand"
	"os"
	"strconv"
	"time"
)

type LossEntry struct {
	offset      time.Duration
	probability float64
}

type LossEmulator struct {
	baseTime          time.Time
	lossEntries       []LossEntry
	currentEntryIndex int
}

func NewLossEmulator(baseTime time.Time, trace string) *LossEmulator {
	return &LossEmulator{
		baseTime:          baseTime,
		lossEntries:       loadLossTrace(trace),
		currentEntryIndex: 0,
	}
}

func loadLossTrace(filename string) []LossEntry {
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	lossTraceReader := csv.NewReader(file)
	var lossEntries []LossEntry
	for {
		row, err := lossTraceReader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}

		offset, err := strconv.Atoi(row[0])
		probability, err := strconv.ParseFloat(row[1], 32)
		lossEntries = append(lossEntries, LossEntry{offset: time.Millisecond * time.Duration(offset), probability: probability})
	}
	return lossEntries
}

func (le *LossEmulator) updateLossEntries(arrivalTime time.Time) {
	nextEntryIndex, nextBase := le.nextEntry()
	nextEntry := le.lossEntries[nextEntryIndex]
	for arrivalTime.After(nextBase.Add(nextEntry.offset)) {
		le.currentEntryIndex = nextEntryIndex
		le.baseTime = nextBase

		nextEntryIndex, nextBase = le.nextEntry()
		nextEntry = le.lossEntries[nextEntryIndex]
	}
}

func (le *LossEmulator) nextEntry() (int, time.Time) {
	if le.currentEntryIndex == len(le.lossEntries)-2 {
		return 0, le.baseTime.Add(le.lossEntries[len(le.lossEntries)-1].offset)

	} else {
		return le.currentEntryIndex + 1, le.baseTime
	}
}

func (le *LossEmulator) lossProbability() float64 {
	return le.lossEntries[le.currentEntryIndex].probability
}

func (le *LossEmulator) Drop(arrivalTime time.Time) bool {
	le.updateLossEntries(arrivalTime)
	return rand.Float64() < le.lossProbability()

}
