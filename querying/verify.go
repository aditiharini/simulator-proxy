package querying

import (
	"bufio"
	"os"
	"strconv"
)

func verify(trace string) {
	traceFile, err := os.Open(trace)
	if err != nil {
		panic(err)
	}
	traceScanner := bufio.NewScanner(traceFile)
	prev := 0
	for traceScanner.Scan() {
		val, err := strconv.Atoi(traceScanner.Text())
		if err != nil {
			panic(err)
		}
		if val < prev {
			panic("Out of order values")
		}
	}
}
