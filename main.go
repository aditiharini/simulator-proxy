package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"time"

	. "github.com/aditiharini/simulator-proxy/simulation"
	"github.com/songgao/water"
)

func readConfig(filename string) Config {
	var config Config
	confFile, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer confFile.Close()
	err = json.NewDecoder(confFile).Decode(&config)
	if err != nil {
		panic(err)
	}
	return config
}

// TODO(aditi): Make this easier to change. This is rigid and ugly
type TopologyJson = map[string](map[string](map[string]interface{}))

func parseTopologyConfig(filename string) TopologyJson {
	var topology TopologyJson
	confFile, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	err = json.NewDecoder(confFile).Decode(&topology)
	if err != nil {
		panic(err)
	}
	return topology
}

// TODO(aditi) : Make this config parsing cleaner
// It would be nice to automatically read into structs
// instead of manually doing casting work
func toLinkConfigs(rawTopology TopologyJson) []LinkConfig {
	var linkConfigs []LinkConfig
	for strSrc, linksByDst := range rawTopology {
		src, err := strconv.Atoi(strSrc)
		if err != nil {
			panic(err)
		}
		for strDst, linkInfo := range linksByDst {
			var dst Address
			if strDst == "base" {

			} else {
				dst, err = strconv.Atoi(strDst)
				if err != nil {
					panic(err)
				}
			}

			var newLinkConfig LinkConfig
			if linkInfo["type"] == "delay" {
				newLinkConfig = NewDelayLinkConfig(
					time.Millisecond*time.Duration(linkInfo["delay"].(float64)),
					src,
					dst,
				)
			} else if linkInfo["type"] == "trace" {
				newLinkConfig = NewTraceLinkConfig(
					linkInfo["filename"].(string),
					src,
					dst,
				)
			} else {
				panic("unsupported link type provided")
			}
			linkConfigs = append(linkConfigs, newLinkConfig)
		}
	}
	return linkConfigs
}

type Config struct {
	NumAddresses        int    `json:"numAddresses"`
	RealSrcAddress      string `json:"realSrcAddress"`
	SimulatedSrcAddress int    `json:"simulatedSrcAddress"`
	SimulatedDstAddress int    `json:"simulatedDstAddress"`
	MaxQueueLength      int    `json:"maxQueueLength"`
	MaxHops             int    `json:"maxHops"`
	DevName             string `json:"devName"`
	DevSrcAddr          string `json:"devSrcAddr"`
	DevDstAddr          string `json:"devDstAddr"`
	RoutingTableNum     string `json:"routingTableNum"`
}

func main() {
	config := readConfig("./config/simulator.json")
	topology := parseTopologyConfig("./config/topology.json")
	dev, err := water.NewTUN(config.DevName)

	if err != nil {
		panic(err)
	}

	exec.Command("ip", "rule", "delete", "table", config.RoutingTableNum).Run()
	exec.Command("ip", "link", "set", "dev", dev.Name(), "up").Run()
	exec.Command("ip", "addr", "add", config.DevSrcAddr, "dev", dev.Name()).Run()
	exec.Command("ip", "rule", "add", "from", config.RealSrcAddress, "table", "1").Run()
	exec.Command("ifconfig", dev.Name(), config.DevSrcAddr, "dstaddr", config.DevDstAddr).Run()
	exec.Command("ip", "route", "add", "default", "dev", dev.Name(), "table", config.RoutingTableNum).Run()

	sim := NewBroadcastSimulator(config.SimulatedDstAddress, dev, net.ParseIP(config.DevDstAddr))
	linkConfigs := toLinkConfigs(topology)

	// Start all link emulation and start receiving/sending packets
	sim.Start(linkConfigs, config.MaxQueueLength)

	packet := make([]byte, 2000)
	for {
		n, err := dev.Read(packet)
		if err != nil {
			panic(err)
		}

		fmt.Printf("packet in %d\n", n)

		packet := Packet{
			Src:         config.SimulatedSrcAddress,
			HopsLeft:    config.MaxHops,
			Data:        packet[:n],
			ArrivalTime: time.Now(),
		}
		sim.BroadcastPacket(packet, config.SimulatedSrcAddress)
	}

}
