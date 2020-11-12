package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
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
			linkInfoMap := linkInfo.(map[string]interface{})
			if linkInfoMap["type"] == "delay" {
				newLinkConfig = NewDelayLinkConfig(
					time.Millisecond*time.Duration(linkInfoMap["delay"].(float64)),
					src,
					dst,
				)
			} else if linkInfoMap["type"] == "trace" {
				newLinkConfig = NewTraceLinkConfig(
					linkInfoMap["file"].(string),
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

func Start(config config.Config, ctx context.Context) {
	// Run sudo sysctl -w net.ipv6.conf.default.accept_ra=0 before
	// starting any mahimahi instances or simulator.
	// This will stop router advertisement messages.
	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: time.StampMicro,
	})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.ErrorLevel)

	devConfig := water.Config{
		DeviceType: water.TUN,
	}
	devConfig.Name = config.General.DevName
	dev, err := water.New(devConfig)

	if err != nil {
		panic(err)
	}

	if err := exec.Command("ip", "rule", "delete", "table", config.General.RoutingTableNum).Run(); err != nil {
		fmt.Println("No iptable deleted")
	}
	if err := exec.Command("ip", "link", "set", "dev", dev.Name(), "up").Run(); err != nil {
		fmt.Println("Cmd: ", "ip link set dev", dev.Name(), "up")
		panic(err)
	}
	if err := exec.Command("ip", "addr", "add", config.General.DevSrcAddr, "dev", dev.Name()).Run(); err != nil {
		fmt.Println("Cmd: ", "ip addr add", config.General.DevSrcAddr, "dev", dev.Name())
		panic(err)
	}
	if err := exec.Command("ip", "rule", "add", "from", config.General.RealSrcAddress, "table", config.General.RoutingTableNum).Run(); err != nil {
		fmt.Println("Cmd: ", "ip rule add from", config.General.RealSrcAddress, "table", config.General.RoutingTableNum)
		panic(err)
	}
	if err := exec.Command("ifconfig", dev.Name(), config.General.DevSrcAddr, "dstaddr", config.General.DevDstAddr).Run(); err != nil {
		fmt.Println("Cmd: ", "ifconfig", dev.Name(), config.General.DevSrcAddr, "dstaddr", config.General.DevDstAddr)
		panic(err)
	}
	if err := exec.Command("ip", "route", "add", "default", "dev", dev.Name(), "table", config.General.RoutingTableNum).Run(); err != nil {
		fmt.Println("Cmd: ", "ip route add default dev", dev.Name(), "table", config.General.RoutingTableNum)
		panic(err)
	}

	sim := NewSimulator(config.General.SimulatedDstAddress, dev, net.ParseIP(config.General.DevDstAddr))
	linkConfigs := toLinkConfigs(config.Topology, config.General.SimulatedDstAddress)

	neighborMap := ToNeighborsMap(linkConfigs)

	// Start all link emulation and start receiving/sending packets
	if config.General.RoutingAlgorithm.Type == "broadcast" {
		sim.SetRouter(NewBroadcastSimulator(neighborMap))
	} else if config.General.RoutingAlgorithm.Type == "best_neighbor" {
		sim.SetRouter(NewBestNeighborSimulator(neighborMap, config.General.SimulatedDstAddress, time.Millisecond*time.Duration(config.General.RoutingAlgorithm.UpdateLag)))
	} else if config.General.RoutingAlgorithm.Type == "oracle" {
		sim.SetRouter(NewOracleRouter(neighborMap))
	} else {
		panic("No valid routing set")
	}
	sim.Start(linkConfigs, config.General.MaxQueueLength)

	id := 0
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Context closed")
			return
		default:
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

			packet := DataPacket{
				Src:         config.General.SimulatedSrcAddress,
				HopsLeft:    config.General.MaxHops,
				Data:        packetData,
				ArrivalTime: time.Now(),
				Id:          id,
			}
			id++
			sim.WriteNewPacket(&packet, packet.Src)
		}
	}
}

func main() {
	// Run sudo sysctl -w net.ipv6.conf.default.accept_ra=0 before
	// starting any mahimahi instances or simulator.
	// This will stop router advertisement messages.
	configFile := flag.String("config", "../../config/simulator/default.json", "some global configuration params")
	runTime := flag.Int("time", -1, "Time to run sim for in seconds")
	flag.Parse()
	config := readConfig(*configFile)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*runTime))
	Start(config, ctx)
	cancel()
}
