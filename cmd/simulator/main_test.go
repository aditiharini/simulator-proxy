package main

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"syscall"
	"testing"
	"time"

	config "github.com/aditiharini/simulator-proxy/config/simulator"
)

var topology = map[string]map[string]interface{}{
	"0": {
		"1": map[string]interface{}{
			"type":  "delay",
			"delay": 1.,
		},
		"2": map[string]interface{}{
			"type":  "delay",
			"delay": 1.,
		},
		"base": map[string]interface{}{
			"type": "trace",
			"file": "/home/ubuntu/simulator-proxy/cmd/experiment/data/trace-2.pps",
		},
	},
	"1": {
		"0": map[string]interface{}{
			"type":  "delay",
			"delay": 1.,
		},
		"2": map[string]interface{}{
			"type":  "delay",
			"delay": 1.,
		},
		"base": map[string]interface{}{
			"type": "trace",
			"file": "/home/ubuntu/simulator-proxy/cmd/experiment/data/trace-3.pps",
		},
	},
	"2": {
		"0": map[string]interface{}{
			"type":  "delay",
			"delay": 1.,
		},
		"1": map[string]interface{}{
			"type":  "delay",
			"delay": 1.,
		},
		"base": map[string]interface{}{
			"type": "trace",
			"file": "/home/ubuntu/simulator-proxy/cmd/experiment/data/trace-2.pps",
		},
	},
}
var general = config.GeneralConfig{
	RealSrcAddress:      "100.64.0.4",
	SimulatedSrcAddress: 0.,
	SimulatedDstAddress: 999.,
	MaxQueueLength:      5000.,
	MaxHops:             2.,
	DevName:             "proxy",
	DevSrcAddr:          "10.0.0.1",
	DevDstAddr:          "10.0.0.2",
	RoutingTableNum:     "1",
	RoutingAlgorithm:    config.RouterConfig{},
}

func CreateConfig(routerConfig config.RouterConfig) config.Config {
	general.RoutingAlgorithm = routerConfig
	return config.Config{Topology: topology, General: general}
}

func run(cmdStr string, tag string, printStdout bool, printStderr bool) *exec.Cmd {
	cmd := exec.Command("bash", "-c", cmdStr)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		panic(err)
	}

	if printStdout {
		go func() {
			buf := bufio.NewReader(stdout)
			for {
				line, _, err := buf.ReadLine()
				if err != nil {
					break
				}
				fmt.Println(tag, string(line))
			}
		}()
	}

	if printStderr {
		go func() {
			buf := bufio.NewReader(stderr)
			for {
				line, _, err := buf.ReadLine()
				if err != nil {
					break
				}
				fmt.Println(tag, string(line))
			}
		}()
	}

	return cmd

}

func RunSetup(addr string) {
	time.Sleep(2 * time.Second)
}

func TestBroadcast(t *testing.T) {
	simConfig := CreateConfig(config.RouterConfig{Type: "broadcast"})
	receiverCmd := fmt.Sprintf("mm-delay 1 iperf -u -s")
	fmt.Println(receiverCmd)
	receiver := run(receiverCmd, "[RECEIVER]", true, true)
	time.Sleep(2 * time.Second)
	simDone := make(chan bool)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(20))
		Start(simConfig, ctx)
		cancel()
		simDone <- true
	}()
	time.Sleep(2 * time.Second)
	senderCmd := fmt.Sprintf("mm-delay 1 iperf -u -b 1M -l 1400 -t 5 -c %s", "100.64.0.2")
	fmt.Println(senderCmd)
	sender := run(senderCmd, "[SENDER]", true, true)
	sender.Wait()
	<-simDone
	syscall.Kill(-receiver.Process.Pid, syscall.SIGTERM)
	fmt.Println("Finished cleanup")
}
