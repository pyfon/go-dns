package main

import (
	"fmt"
	"strconv"
)

type HostList []string
type PortList []int

func (h *HostList) Set(s string) error {
	*h = append(*h, s)
	return nil
}

func (h *HostList) String() string {
	return fmt.Sprint(*h)
}

func (p *PortList) Set(s string) error {
	i, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	*p = append(*p, i)
	return nil
}

func (p *PortList) String() string {
	return fmt.Sprint(*p)
}
