package lib

import (
	"net"
	"strconv"
)

func IsValidHost(host string) bool {
	host, port, err := net.SplitHostPort(host)
	if err != nil {
		return false
	}

	if _, ok := strconv.ParseUint(port, 10, 64); ok != nil {
		return false
	}

	addrs := localAddrs()

	if host == "" {
		return true
	}

	if !isValidIP(host) {
		return false
	}

	if !containsAddr(addrs, host) {
		return false
	}

	return true
}

func localAddrs() []string {
	addrs := make([]string, 0, 3)
	ifaces, _ := net.Interfaces()
	for _, i := range ifaces {
		ipAddrs, _ := i.Addrs()
		for _, addr := range ipAddrs {
			switch v := addr.(type) {
			case *net.IPNet:
				ip := v.IP
				addrs = append(addrs, ip.String())
			case *net.IPAddr:
				ip := v.IP
				addrs = append(addrs, ip.String())
			}
		}
	}
	return addrs
}

func containsAddr(addrs []string, str string) bool {
	for _, v := range addrs {
		if v == str {
			return true
		}
	}
	return false
}

func isValidIP(host string) bool {
	addr := net.ParseIP(host)
	if addr == nil {
		return false
	}
	return true
}

func ContainsHost(nodes []*Node, host string) bool {
	for _, v := range nodes {
		if v.Host == host {
			return true
		}
	}
	return false
}
