package main

import (
	"fmt"
	"net"
	"strconv"
)

type SocketList []net.UDPAddr

func (h *SocketList) Set(s string) error {
	host, portStr, err := net.SplitHostPort(s)
	if err != nil {
		return err
	}
	ips, err := hostToIP(host)
	if err != nil {
		return err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return err
	}
	for _, ip := range ips {
		*h = append(*h, net.UDPAddr{IP: ip, Port: port})
	}
	return nil
}

func (h *SocketList) String() string {
	return fmt.Sprint(*h)
}

func hostToIP(host string) ([]net.IP, error) {
	ip := net.ParseIP(host)
	if ip != nil {
		return []net.IP{ip}, nil
	}
	// host might be a hostname.
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}
	return ips, nil
}
