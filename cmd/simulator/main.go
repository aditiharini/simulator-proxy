package main

import (
	"encoding/json"
	"flag"
	"net"
	"os"
	"os/exec"
	"strconv"
	"time"

	config "github.com/aditiharini/simulator-proxy/config/simulator"
	. "github.com/aditiharini/simulator-proxy/simulation"
	log "github.com/sirupsen/logrus"
	"github.com/songgao/water"
)

func readConfig(filename string) config.Config {
	var config config.Config
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

func parseTopologyConfig(filename string) config.TopologyJson {
	var topology config.TopologyJson
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
func toLinkConfigs(rawTopology config.TopologyJson, simulatedDstAddress Address) []LinkConfig {
	var linkConfigs []LinkConfig
	for strSrc, linksByDst := range rawTopology {
		src, err := strconv.Atoi(strSrc)
		if err != nil {
			panic(err)
		}
		for strDst, linkInfo := range linksByDst {
			var dst Address
			if strDst == "base" {
				dst = simulatedDstAddress
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
					linkInfo["file"].(string),
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

func main() {
	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: time.StampMicro,
	})
	log.SetOutput(os.Stdout)
	// Run sudo sysctl -w net.ipv6.conf.default.accept_ra=0 before
	// starting any mahimahi instances or simulator.
	// This will stop router advertisement messages.
	configFile := flag.String("config", "../../config/simulator/global/default.json", "some global configuration params")
	topologyFile := flag.String("topology", "../../config/topology/default.json", "topology configuration")
	flag.Parse()

	config := readConfig(*configFile)
	topology := parseTopologyConfig(*topologyFile)
	devConfig := water.Config{
		DeviceType: water.TUN,
	}
	devConfig.Name = config.DevName
	dev, err := water.New(devConfig)

	if err != nil {
		panic(err)
	}

	exec.Command("ip", "rule", "delete", "table", config.RoutingTableNum).Run()
	exec.Command("ip", "link", "set", "dev", dev.Name(), "up").Run()
	exec.Command("ip", "addr", "add", config.DevSrcAddr, "dev", dev.Name()).Run()
	exec.Command("ip", "rule", "add", "from", config.RealSrcAddress, "table", config.RoutingTableNum).Run()
	exec.Command("ifconfig", dev.Name(), config.DevSrcAddr, "dstaddr", config.DevDstAddr).Run()
	exec.Command("ip", "route", "add", "default", "dev", dev.Name(), "table", config.RoutingTableNum).Run()

	sim := NewBroadcastSimulator(config.SimulatedDstAddress, dev, net.ParseIP(config.DevDstAddr))
	linkConfigs := toLinkConfigs(topology, config.SimulatedDstAddress)

	// Start all link emulation and start receiving/sending packets
	sim.Start(linkConfigs, config.MaxQueueLength)

	id := 0
	for {
		packetBuf := make([]byte, 2000)
		n, err := dev.Read(packetBuf)
		if err != nil {
			panic(err)
		}
		log.WithFields(log.Fields{
			"event": "packet_received",
			"id":    id,
		}).Info()
		packetData := packetBuf[:n]

		packet := Packet{
			Src:         config.SimulatedSrcAddress,
			HopsLeft:    config.MaxHops,
			Data:        packetData,
			ArrivalTime: time.Now(),
			Id:          id,
		}
		id++
		sim.BroadcastPacket(packet, config.SimulatedSrcAddress)
	}

}
