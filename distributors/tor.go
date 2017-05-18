package distributors

import (
	"crypto/tls"
	"fmt"
	"golang.org/x/net/proxy"
	"io/ioutil"
	"net/http"
	"os/exec"
	"time"
)

type Tor struct {
	Clients  *ClientList
	Count    int
	MaxCount int
	Used     int
}

func NewTor(daemons, count, max, headerTimeout, requestTimeout int) *Tor {
	startSocksPort := 9150
	availableDaemons := daemons
	run("kill", "$(pgrep tor)")

	Clients := NewClientList()
	for i := 0; i < availableDaemons; i++ {
		addr := fmt.Sprintf("%v", startSocksPort+i)
		addr2 := fmt.Sprintf("%v", startSocksPort+i+availableDaemons)
		dir := fmt.Sprintf("/progress/tor_dir/tor%v", i)

		run("mkdir", "-p", dir)
		err := run("tor",
			"--RunAsDaemon", "1",
			"--SocksPort", addr,
			"--ControlPort", addr2,
			"--DataDirectory", dir,

			// Random password to disable control port access
			"--HashedControlPassword", "16:118E516CCAA79CF76014434BD85092BE8E34C6D0D7594C2F5D4093F78B",

			// Disable routing
			"--ClientOnly", "1",
			//"--MaxCircuitDirtiness", "300", // Maximum seconden om tor circuit te hergebruiken
			//"--OnionTrafficOnly", "1", (unsupported)
			//"--SafeSocks", "1", // Voorkom dns leaks (aanvragen met al geresolvede dns worden genegeerd)
		)

		//tor --RunAsDaemon 1 --SocksPort 9150 --ControlPort 9180 --DataDirectory "/tor_dir/tor1" --HashedControlPassword "16:118E516CCAA79CF76014434BD85092BE8E34C6D0D7594C2F5D4093F78B" --ClientOnly 1

		if err != nil {
			fmt.Println("ERROR LAUNCHING tor: " + err.Error())
		}

		torDialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("127.0.0.1:%v", addr), nil, proxy.Direct)

		if err != nil {
			fmt.Println("Proxy error" + err.Error())
		}

		transport := &http.Transport{
			Dial:         torDialer.Dial,
			MaxIdleConns: 500,
			//DisableKeepAlives: true, // Hmmm?
			/*TLSHandshakeTimeout:   10 * time.Second,
			  MaxIdleConnsPerHost:   0,
			  ResponseHeaderTimeout: 10 * time.Second,*/
			ResponseHeaderTimeout: time.Duration(headerTimeout) * time.Second,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true}, // Onveilige https toelaten
		}
		Clients.Push(&http.Client{
			Transport: transport,
			Timeout:   time.Duration(requestTimeout) * time.Second,
		})
	}

	// Wachten
	time.Sleep(10 * time.Second)

	return &Tor{Clients: Clients, Count: count, MaxCount: max}
}

func (dist *Tor) GetClient() *http.Client {
	if dist.Used >= dist.Count {
		return nil
	}
	dist.Used++

	client := dist.Clients.Pop()
	dist.Clients.Push(client)

	return client
}

func (dist *Tor) FreeClient(client *http.Client) {
	dist.Used--
}

func (dist *Tor) DecreaseClients() {
	if dist.Count < 10 {
		return
	}
	dist.Count = int(float64(dist.Count) * 0.8)
}

func (dist *Tor) IncreaseClients() {
	dist.Count = int(float64(dist.Count) * 1.05)
	if dist.Count > dist.MaxCount {
		dist.Count = dist.MaxCount
	}
}

func (dist *Tor) AvailableClients() int {
	return dist.Count - dist.Used
}

func (dist *Tor) UsedClients() int {
	return dist.Used
}

func run(command string, arguments ...string) error {
	cmd := exec.Command(command, arguments...)

	// Connect pipe to read Stderr
	stderr, err := cmd.StderrPipe()

	if err != nil {
		// Failed to connect pipe
		return fmt.Errorf("%q failed to connect stderr pipe: %v", command, err)
	}

	stdout, err := cmd.StdoutPipe()

	if err != nil {
		// Failed to connect pipe
		return fmt.Errorf("%q failed to connect stdout pipe: %v", command, err)
	}

	// Do not use cmd.Run()
	if err := cmd.Start(); err != nil {
		// Problem while copying stdin, stdout, or stderr
		return fmt.Errorf("%q failed: %v", command, err)
	}

	// Zero exit status
	// Darwin: launchctl can fail with a zero exit status,
	// so check for emtpy stderr
	slurp, _ := ioutil.ReadAll(stderr)
	slurpout, _ := ioutil.ReadAll(stdout)

	if len(slurp) > 0 {
		return fmt.Errorf("%q failed with stderr: %s", command, slurp)
	}

	if err := cmd.Wait(); err != nil {
		// Command didn't exit with a zero exit status.
		return fmt.Errorf("%q failed with exit status: %v, %s", command, err, slurpout)
	}

	return nil
}
