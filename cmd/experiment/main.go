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
)

// Only have drone to base station
// Generate config that sets drone to drone links
type FullyConnectedJson = map[string]string

type ConfigEntry interface {
}

type DelayEntry struct {
	EntryType   string `json:"type"`
	DelayMillis int    `json:"delay"`
}

func NewDelayEntry(delayMillis int) DelayEntry {
	return DelayEntry{EntryType: "delay", DelayMillis: delayMillis}
}

type TraceEntry struct {
	EntryType string `json:"type"`
	TraceFile string `json:"file"`
}

func NewTraceEntry(tracefile string) TraceEntry {
	return TraceEntry{EntryType: "trace", TraceFile: tracefile}
}

func writeGeneralConfig(topology FullyConnectedJson, outputDir string) {
	generalTopology := make(map[string](map[string]interface{}))
	for strSrc, trace := range topology {
		generalTopology[strSrc] = make(map[string]interface{})
		generalTopology[strSrc]["base"] = NewTraceEntry(trace)

		for strDst, _ := range topology {
			if strSrc != strDst {
				generalTopology[strSrc][strDst] = NewDelayEntry(1)
			}
		}
	}
	data, err := json.Marshal(generalTopology)
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

func writeLinkConfigs(topology FullyConnectedJson, outputDir string) {
	for strSrc, trace := range topology {
		generalTopology := make(map[string](map[string]interface{}))
		generalTopology["0"] = make(map[string]interface{})
		generalTopology["0"]["base"] = NewTraceEntry(trace)

		linkFile, err := os.Create(fmt.Sprintf("%s/%s.json", outputDir, strSrc))
		if err != nil {
			panic(err)
		}
		data, err := json.Marshal(generalTopology)
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

func processConfig(filename string, fullDir string, linksDir string) {
	var topology FullyConnectedJson
	confFile, err := os.Open(filename)
	if err != nil {
		panic(err)
	}

	defer confFile.Close()

	err = json.NewDecoder(confFile).Decode(&topology)
	if err != nil {
		panic(err)
	}

	writeGeneralConfig(topology, fullDir)
	writeLinkConfigs(topology, linksDir)

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

func runSimulator(inputFile string, outputFile string) {
	receiver := run("cd ../packet-receiver && mm-delay 1 ./packet-receiver -listen-on=100.64.0.2:8080", "RECV", false, true)
	time.Sleep(time.Second * time.Duration(1))

	path := fmt.Sprintf("%s/%s", "../experiment", inputFile)
	outpath := fmt.Sprintf("%s/%s", "../experiment", outputFile)
	simCmd := fmt.Sprintf("cd ../simulator && sudo ./simulator -topology=%s > %s", path, outpath)
	run(simCmd, "SIM", true, true)
	time.Sleep(time.Second * time.Duration(1))

	out, err := exec.Command("bash", "-c", "cd ../packet-sender && mm-delay 1 ./packet-sender -dest=100.64.0.2:8080 -count=5000 -size=1000 -wait=1").CombinedOutput()
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
	fullyConnectedConfig := flag.String("topology", "", "fully connected config")
	experimentName := flag.String("experimentName", "", "name to upload experiment with")
	flag.Parse()

	exec.Command("rm", "-rf", "tmp").Run()
	exec.Command("mkdir", "tmp").Run()
	exec.Command("mkdir", "-p", "tmp/inputs/links").Run()
	exec.Command("mkdir", "tmp/inputs/full").Run()
	exec.Command("mkdir", "-p", "tmp/outputs/links").Run()
	exec.Command("mkdir", "tmp/outputs/full").Run()
	exec.Command("mkdir", "tmp/outputs/csv").Run()
	processConfig(*fullyConnectedConfig, "tmp/inputs/full", "tmp/inputs/links")

	linkFiles, err := ioutil.ReadDir("tmp/inputs/links")
	if err != nil {
		panic(err)
	}

	for _, file := range linkFiles {
		inpath := fmt.Sprintf("%s/%s", "tmp/inputs/links", file.Name())
		outpath := fmt.Sprintf("%s/%s.log", "tmp/outputs/links", strings.Split(file.Name(), ".")[0])
		runSimulator(inpath, outpath)
		time.Sleep(time.Second * time.Duration(1))
	}

	runSimulator("tmp/inputs/full/full.json", "tmp/outputs/full/full.log")

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
