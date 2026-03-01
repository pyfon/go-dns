package main

import (
	"fmt"
)

type SocketList []string

func (h *SocketList) Set(s string) error {
	*h = append(*h, s)
	return nil
}

func (h *SocketList) String() string {
	return fmt.Sprint(*h)
}
