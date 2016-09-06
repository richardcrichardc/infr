package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

var config struct {
	General        map[string]string
	Keys           string
	Hosts          []*host
	Lxcs           []*lxc
	Backups        []*backup
	LastPreseedURL string
}

var hostsDown string

func loadConfig() {
	cdWorkDir()
	defer restoreCwd()

	jsonBytes, err := ioutil.ReadFile("config")
	if err != nil {
		if os.IsNotExist(err) {
			// No config
			config.General = make(map[string]string)
			return
		}
		errorExit("Error reading config: %s", err)
	}

	loadConfig2(jsonBytes)
}

func loadConfig2(configBytes []byte) {
	// Parse config
	err := json.Unmarshal(configBytes, &config)

	if err != nil {
		errorExit("Error parsing config: %s\n%s", err, string(configBytes))
	}

	// Mark hosts from -D switch as down
	for _, hostname := range strings.Split(hostsDown, ",") {
		for _, h := range config.Hosts {
			if hostname == h.Name {
				h.down = true
			}
		}
	}

	// Check that everyone has the same config
	c := make(chan string)
	hostCount := 0
	for _, h := range config.Hosts {
		if !h.down {
			hostCount += 1
			go func(h *host) {
				h.MustConnectSSH()
				c <- h.SudoCaptureStdout("cat /etc/infr-config || echo ''")
			}(h)
		}
	}

	var remoteConfig string
	first := true
	for i := 0; i < hostCount; i++ {
		conf := <-c

		if first {
			remoteConfig = conf
			first = false
		} else {
			if conf != remoteConfig {
				errorExit("Copies of config stored in /etc/infr-config of hosts differ. This must be fixed manually.")
			}
		}
	}

	remoteConfigBytes := []byte(remoteConfig)
	if len(remoteConfigBytes) != 0 && !bytes.Equal(configBytes, remoteConfigBytes) {
		// Remote configs match each other but differ from local config, someone else has changed something.
		// Reload remote configs in case another host has been added (this is the uncommon worst case and can be optimised)
		fmt.Println("Remote config differs, reloading in case hosts have been added")

		for _, h := range config.Hosts {
			if h.sshClient != nil {
				go h.sshClient.Close()
			}
		}
		loadConfig2(remoteConfigBytes)

		// Save localy to avoid reload if action does not save
		saveConf(false, false)
	}
}

func saveConfig() {
	saveConf(true, true)
}

func saveConf(saveToHosts, saveToDownHosts bool) {
	cdWorkDir()
	defer restoreCwd()

	jsonBytes, err := json.MarshalIndent(config, "", "  ")

	if err != nil {
		errorExit("Error in saveConfig: %s", err)
	}

	err = ioutil.WriteFile("config", jsonBytes, 0600)

	if err != nil {
		errorExit("Error in saveConfig: %s", err)
	}

	if !saveToHosts {
		return
	}

	// Save copy of config on all hosts
	jsonStr := string(jsonBytes)
	c := make(chan bool)
	hostCount := len(config.Hosts)

	for _, h := range config.Hosts {
		go func(h *host) {
			if h.down {
				if !saveToDownHosts {
					c <- true
					return
				}

				if h.ConnectSSH() != nil || !h.Lock() {
					fmt.Println("Unable to save config to", h.Name)
					c <- true
					return
				}
			}

			h.Upload(jsonStr, "/etc/infr-config")
			c <- true
		}(h)
	}

	for i := 0; i < hostCount; i++ {
		<-c
	}
}
