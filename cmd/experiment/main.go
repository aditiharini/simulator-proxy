package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	config "github.com/aditiharini/simulator-proxy/config/experiment"
	simulatorConfig "github.com/aditiharini/simulator-proxy/config/simulator"
)

// Only have drone to base station
// Generate config that sets drone to drone links

func writeGeneralConfig(simConfig config.SimulatorConfig, outputDir string) {
	generalTopology := make(map[string](map[string]interface{}))
	for strSrc, trace := range simConfig.Topology {
		generalTopology[strSrc] = make(map[string]interface{})
		generalTopology[strSrc]["base"] = config.NewTraceEntry(trace)

		for strDst, _ := range simConfig.Topology {
			if strSrc != strDst {
				generalTopology[strSrc][strDst] = config.NewDelayEntry(1)
			}
		}
	}
	generalConfig := simulatorConfig.Config{Topology: generalTopology, General: simConfig.Global}
	data, err := json.Marshal(generalConfig)
	if err != nil {
		panic(err)
	}

	outFile, err := os.Create(fmt.Sprintf("%s/full.json", outputDir))
	if err != nil {
		panic(err)
	}

	_, err = outFile.Write(data)
	if err != nil {
		panic(err)
	}
	defer outFile.Close()
}

func writeLinkConfigs(simConfig config.SimulatorConfig, outputDir string) {
	for strSrc, trace := range simConfig.Topology {
		generalTopology := make(map[string](map[string]interface{}))
		generalTopology["0"] = make(map[string]interface{})
		generalTopology["0"]["base"] = config.NewTraceEntry(trace)

		linkFile, err := os.Create(fmt.Sprintf("%s/%s.json", outputDir, strSrc))
		if err != nil {
			panic(err)
		}
		generalConfig := simulatorConfig.Config{Topology: generalTopology, General: simConfig.Global}
		data, err := json.Marshal(generalConfig)
		if err != nil {
			panic(err)
		}

		_, err = linkFile.Write(data)
		if err != nil {
			panic(err)
		}
		defer linkFile.Close()
	}
}

func processConfig(filename string, fullDir string, linksDir string) config.Config {
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

	writeGeneralConfig(config.Simulator, fullDir)
	writeLinkConfigs(config.Simulator, linksDir)
	return config
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

func runSimulator(config config.Config, inputFile string, outputFile string) {
	receiverCmd := fmt.Sprintf("cd ../packet-receiver && mm-delay 1 ./packet-receiver -listen-on=%s", config.Receiver.Address)
	receiver := run(receiverCmd, "RECV", false, true)
	time.Sleep(time.Second * time.Duration(1))

	inpath := fmt.Sprintf("%s/%s", "../experiment", inputFile)
	outpath := fmt.Sprintf("%s/%s", "../experiment", outputFile)
	simCmd := fmt.Sprintf("cd ../simulator && sudo ./simulator -config=%s > %s", inpath, outpath)
	run(simCmd, "SIM", true, true)
	time.Sleep(time.Second * time.Duration(1))

	senderCmd := fmt.Sprintf(
		"cd ../packet-sender && mm-delay 1 ./packet-sender -dest=%s -count=%d -size=%d -wait=%d",
		config.Receiver.Address,
		config.Sender.Count,
		config.Sender.Size,
		config.Sender.Wait,
	)
	if config.Sender.Traffic == "bursty" {
		senderCmd = fmt.Sprintf("%s -bursty -packetsPerBurst=%d", senderCmd, config.Sender.PacketsPerBurst)
	}

	out, err := exec.Command("bash", "-c", senderCmd).CombinedOutput()
	if err != nil {
		fmt.Println(string(out))
		panic(err)
	}

	// Give ample time for the transmission to finish
	time.Sleep(time.Second * time.Duration(10))

	// Something goes wrong with killing simulator. I think it's because of sudo.
	// This is a hack around that
	if err := exec.Command("sudo", "killall", "simulator").Run(); err != nil {
		panic(err)
	}
	syscall.Kill(-receiver.Process.Pid, syscall.SIGTERM)
	time.Sleep(time.Second * time.Duration(1))
}

func main() {
	// Generate config- first hard code, then take parameters
	// This is necessary when trying to increase number of drones

	// Create logs for individual runs and for full simulation run
	// Store experiment results in dropbox

	// Need to be able to generate traces in the future
	fullyConnectedConfig := flag.String("config", "", "fully connected config")
	experimentName := flag.String("experimentName", "", "name to upload experiment with")
	flag.Parse()

	exec.Command("rm", "-rf", "tmp").Run()
	exec.Command("mkdir", "tmp").Run()
	exec.Command("mkdir", "-p", "tmp/inputs/links").Run()
	exec.Command("mkdir", "tmp/inputs/full").Run()
	exec.Command("mkdir", "-p", "tmp/outputs/links").Run()
	exec.Command("mkdir", "tmp/outputs/full").Run()
	exec.Command("mkdir", "tmp/outputs/csv").Run()
	config := processConfig(*fullyConnectedConfig, "tmp/inputs/full", "tmp/inputs/links")

	linkFiles, err := ioutil.ReadDir("tmp/inputs/links")
	if err != nil {
		panic(err)
	}

	for _, file := range linkFiles {
		inpath := fmt.Sprintf("%s/%s", "tmp/inputs/links", file.Name())
		outpath := fmt.Sprintf("%s/%s.log", "tmp/outputs/links", strings.Split(file.Name(), ".")[0])
		runSimulator(config, inpath, outpath)
		time.Sleep(time.Second * time.Duration(1))
	}

	runSimulator(config, "tmp/inputs/full/full.json", "tmp/outputs/full/full.log")

	linkLogs, err := ioutil.ReadDir("tmp/outputs/links")
	var strLinkLogs []string
	for _, log := range linkLogs {
		strLinkLogs = append(strLinkLogs, fmt.Sprintf("../experiment/tmp/outputs/links/%s", log.Name()))
	}

	fullLinkLogs := strings.Join(strLinkLogs, ",")
	fullLog := "../experiment/tmp/outputs/full/full.log"
	logCmd := fmt.Sprintf("cd ../process-logs && ./process-logs -newlog=%s -linkLogs=%s -outdir=%s", fullLog, fullLinkLogs, "../experiment/tmp/outputs/csv")
	if out, err := exec.Command("bash", "-c", logCmd).CombinedOutput(); err != nil {
		fmt.Println(string(out))
		panic(err)
	}

	if *experimentName != "" {
		uploadCmd := fmt.Sprintf("~/dropbox_uploader.sh upload tmp Drone-Project/broadcast/%s", *experimentName)
		if out, err := exec.Command("bash", "-c", uploadCmd).CombinedOutput(); err != nil {
			fmt.Println(string(out))
			panic(err)
		}
	}

}
