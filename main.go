package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/toxyl/flo"
	"github.com/toxyl/glog"
)

var (
	log = glog.NewLoggerSimple("TorMan")
)

func init() {
	glog.LoggerConfig.ShowDateTime = false
	glog.LoggerConfig.ShowIndicator = true
	glog.LoggerConfig.ShowRuntimeMilliseconds = false
	glog.LoggerConfig.ShowRuntimeSeconds = false
	glog.LoggerConfig.ShowSubsystem = false
	glog.LoggerConfig.SplitOnNewLine = true
	glog.LoggerConfig.CheckIfURLIsAlive = false
}

func systemctl(op string, port int) {
	if port == 0 {
		exec.Command("systemctl", op).Run()
		return
	}
	exec.Command("systemctl", op, serviceName(port)).Run()
}

func serviceName(port int) string {
	return fmt.Sprintf("tor-instance-%d.service", port)
}

func servicePath(port int) string {
	return fmt.Sprintf("/etc/systemd/system/%s", serviceName(port))
}

func configPath(port int) string {
	return fmt.Sprintf(`/var/lib/tor/torrc-%d`, port)
}

func instanceDir(port int) string {
	return fmt.Sprintf("/var/lib/tor/instance%d", port)
}

func createTorConfig(port int) {
	dataDir := instanceDir(port)
	configContent := fmt.Sprintf(`SocksPort 0.0.0.0:%d
DataDirectory %s
Log notice file /var/log/tor/notices.log
NumCPUs 1
RelayBandwidthRate 100 MB  # Adjust as per your available bandwidth
RelayBandwidthBurst 200 MB # Adjust as per your available bandwidth
DirCache 1
IPv6Exit 1
MaxMemInQueues 256 MB
CircuitBuildTimeout 30
LearnCircuitBuildTimeout 1
SocksTimeout 60
HardwareAccel 1
ClientRejectInternalAddresses 1
ConnLimit 1024
BandwidthRate 100 MB
BandwidthBurst 200 MB
NewCircuitPeriod 30
UseHelperNodes 1
NumHelperNodes 3
MaxCircuitDirtiness 600
UseEntryGuards 1
UseBridges 0
`, port, dataDir)
	flo.File(configPath(port)).WriteString(configContent).Perm(0644)
}

func createSystemdService(port int) {
	serviceContent := fmt.Sprintf(`[Unit]
Description=Tor instance on port %d
After=network.target

[Service]
User=debian-tor
ExecStart=/usr/bin/tor -f %s
ExecReload=/bin/kill -HUP $MAINPID
KillSignal=SIGINT
Restart=on-failure
LimitNOFILE=8192

[Install]
WantedBy=multi-user.target
`, port, configPath(port))
	flo.File(servicePath(port)).WriteString(serviceContent).Perm(0644)
	systemctl("daemon-reload", 0)
	systemctl("enable", port)
}

func startInstance(port int) {
	createTorConfig(port)
	createSystemdService(port)
	systemctl("start", port)
	log.BlankAuto("Started instance %s", port)
}

func stopInstance(port int) {
	systemctl("stop", port)
	log.BlankAuto("Stopped instance %s", port)
}

func findAllInstances() []int {
	res := []int{}
	d := flo.Dir("/var/lib/tor/")
	d.Each(func(f *flo.FileObj) {
		if strings.HasPrefix(f.Name(), "torrc-") {
			name := strings.TrimPrefix(f.Name(), "torrc-")
			p, err := strconv.Atoi(name)
			if p > 0 {
				res = append(res, p)
			}
			if err != nil {
				log.ErrorAuto("couldn't parse %s: %s", f.Name(), err)
			}
		}
	}, nil)

	return res
}

func start(basePort int) {
	stop() // just in case there are still active instances

	numCores := runtime.NumCPU()
	for i := 0; i < numCores; i++ {
		startInstance(basePort + i)
	}
	log.OKAuto("All Tor instances started.")

	lastPort := basePort + numCores - 1
	log.InfoAuto("Remember to open the ports: ufw allow from %s to any port %s:%s proto tcp", "192.168.1.100", basePort, lastPort)
	log.BlankAuto("Replace %s with the actual IP you want to whitelist.", "192.168.1.100")
}

func stop() {
	for _, port := range findAllInstances() {
		stopInstance(port)
		removeConfig(port)
	}
	log.OKAuto("All Tor instances stopped.")
}

func removeConfig(port int) {
	systemctl("disable", port)
	os.Remove(servicePath(port))
	systemctl("daemon-reload", port)
	os.RemoveAll(instanceDir(port))
	log.InfoAuto("Configuration for instance %s removed.", port)
}

func help() {
	log.BlankAuto("Usage: torman start <base port>")
	log.BlankAuto("       torman stop")
	os.Exit(0)
}

func main() {
	if len(os.Args) < 2 {
		help()
	}

	command := os.Args[1]
	switch command {
	case "start":
		if len(os.Args) < 3 {
			help()
		}
		basePort, _ := strconv.Atoi(os.Args[2])
		start(basePort)
	case "stop":
		stop()
	default:
		log.ErrorAuto("Unknown command: %s", command)
		help()
	}
}
