package crawler

import (
	"crypto/tls"
	"fmt"
	"golang.org/x/net/proxy"
	"io/ioutil"
	"net/http"
	//"os"
	"os/exec"
	"time"
)

type TorDistributor struct {
	Clients          *ClientList
	AvailableDaemons int
}

func NewTorDistributor() *TorDistributor {
	/*e := run("cat", fmt.Sprintf("/proc/%v/limits", os.Getpid()))
	if e != nil {
		fmt.Println(e.Error())
	}*/

	StartSocksPort := 9150
	AvailableDaemons := 20 //35

	daemonList := make([]*http.Client, AvailableDaemons)
	for i := 0; i < AvailableDaemons; i++ {
		addr := fmt.Sprintf("%v", StartSocksPort+i)
		addr2 := fmt.Sprintf("%v", StartSocksPort+i+AvailableDaemons)
		dir := fmt.Sprintf("/progress/tor_dir/tor%v", i)

		run("mkdir", "-p", dir)
		err := run("tor",
			"--RunAsDaemon", "1",
			"--SocksPort", addr,
			"--ControlPort", addr2,
			//"--CookieAuthentication", "0",
			//"--HashedControlPassword", "",
			//"--PidFile", fmt.Sprintf("/progress/tor%v.pid", i),
			"--DataDirectory", dir,
		)

		if err != nil {
			fmt.Println("ERROR LAUNCHING tor: " + err.Error())
		}

		torDialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("127.0.0.1:%v", addr), nil, proxy.Direct)

		if err != nil {
			fmt.Println("Proxy error" + err.Error())
		}

		transport := &http.Transport{
			Dial:         torDialer.Dial,
			MaxIdleConns: 600,
			//DisableKeepAlives: true, // Hmmm?
			/*TLSHandshakeTimeout:   10 * time.Second,
			MaxIdleConnsPerHost:   0,
			ResponseHeaderTimeout: 10 * time.Second,*/
			ResponseHeaderTimeout: 15 * time.Second,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true}, // Onveilige https toelaten
		}
		daemonList[i] = &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		}
	}

	Clients := NewClientList()
	// Beschikbaarheid per proxy
	for k := 0; k < 40; k++ {
		for i := 0; i < AvailableDaemons; i++ {
			Clients.Push(daemonList[i])
		}
	}
	return &TorDistributor{AvailableDaemons: AvailableDaemons, Clients: Clients}
}

func (dist *TorDistributor) GetClient() *http.Client {
	return dist.Clients.Pop()
}

func (dist *TorDistributor) FreeClient(client *http.Client) {
	dist.Clients.Push(client)
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

/*if cfg.TorProxyAddress != nil {
	torDialer, err := proxy.SOCKS5("tcp", *cfg.TorProxyAddress, nil, proxy.Direct)

	if err != nil {
		cfg.LogError(err)
		return nil
	}
	transport = &http.Transport{
		Dial: torDialer.Dial,
	}
} else {
	transport = &http.Transport{}
}

client := &http.Client{Transport: transport, Timeout: time.Second * 10}*/
