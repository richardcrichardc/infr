package main

import (
	"net"
)

func vnetNetwork() *net.IPNet {
	networkStr := needGeneralConfig("vnetNetwork")

	_, network, err := net.ParseCIDR(networkStr)
	if err != nil {
		errorExit("Unable to parse config item 'vnetNetwork': %s", networkStr)
	}

	network.IP = network.IP.To4()
	if network.IP == nil {
		errorExit("Config item 'vnetNetwork' does not support IPv6")
	}

	return network
}

func vnetGetIP() string {

	network := vnetNetwork()

	poolStartStr := needGeneralConfig("vnetPoolStart")
	lastStr := generalConfig("vnetLast")

	poolStart := net.ParseIP(poolStartStr)
	if poolStart == nil {
		errorExit("Unable to parse config item 'vnetPoolStart': %s", poolStartStr)
	}

	poolStart = poolStart.To4()
	if poolStart == nil {
		errorExit("Config item 'vnetPoolStart' does not support IPv6")
	}

	if !network.Contains(poolStart) {
		errorExit("Config item 'vnetPoolStart' must be in network 'vnetNetwork'")
	}

	var last net.IP

	if lastStr != "" {
		last = net.ParseIP(lastStr)
		if last == nil {
			errorExit("Unable to parse config item 'vnetLast': %s", lastStr)
		}

		last = last.To4()
		if last == nil {
			errorExit("Config item 'vnetNext' does not support IPv6")
		}

	} else {
		last = poolStart
	}

	usedIPs := assignedIPs()

incrIPLoop:
	// increment IP
	last[3] += 1
	if last[3] == 0 {
		last[2] += 1
		if last[2] == 0 {
			last[1] += 1
			if last[1] == 0 {
				last[0] += 1
				if last[0] == 0 {
					last = net.IPv4(0, 0, 0, 0)
				}
			}
		}
	}

	// if outside subnet, loop back to start
	if !network.Contains(last) {
		last = poolStart
	}

	lastStr = last.String()

	// check if IP is assigned to another machine
	for _, usedIP := range usedIPs {
		if lastStr == usedIP {
			goto incrIPLoop
		}
	}

	config.General["vnetLast"] = lastStr

	return lastStr
}

func assignedIPs() []string {
	var ips []string

	for _, host := range config.Hosts {
		ips = append(ips, host.PrivateIPv4)
		ips = append(ips, host.BridgeIPv4)
	}

	for _, lxc := range config.Lxcs {
		ips = append(ips, lxc.PrivateIPv4)
	}

	return ips
}
