package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	. "github.com/aditiharini/simulator-proxy/simulation"
)

type Stats struct {
	numWins map[Address]int
}

func (s *Stats) prettyPrint() {
	for address, numWins := range s.numWins {
		fmt.Printf("Address %v won %v times\n", address, numWins)
	}
}

func parseLine(line string) (address Address, id int) {
	splitLine := strings.Split(line, " ")
	id, err := strconv.Atoi(splitLine[1])
	if err != nil {
		panic(err)
	}
	address, err = strconv.Atoi(splitLine[3])
	if err != nil {
		panic(err)
	}
	return address, id
}

func main() {
	filename := os.Args[1]
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	stats := Stats{numWins: make(map[Address]int)}
	for scanner.Scan() {
		address, _ := parseLine(scanner.Text())
		if _, ok := stats.numWins[address]; ok {
			stats.numWins[address] = 1
		} else {
			stats.numWins[address]++
		}
	}

	stats.prettyPrint()
}
