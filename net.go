package main

import (
	"context"
	"fmt"
	"net"

	log "github.com/sirupsen/logrus"
)

type SocketList []net.UDPAddr

// Serve DNS on the given socket until program termination.
func Serve(sock net.UDPAddr, zones Trie[Zone], ctx context.Context) error {
	conn, err := net.ListenUDP("udp", &sock)
	if err != nil {
		log.Errorf("Could not serve on socket %v: %v", sock, err)
		return err
	}
	defer conn.Close()
	log.Infof("Serving DNS on %v:%v", sock.IP, sock.Port)

	for {
		buf := make([]byte, 512)
		n, raddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Errorf("Could not read UDP request: %v", err)
			continue
		}
		go Handle(conn, raddr, buf[:n], &zones)
	}
}

// Handle will handle a connection.
func Handle(conn *net.UDPConn, raddr *net.UDPAddr, query []byte, zones *Trie[Zone]) {
	logHead := fmt.Sprintf("[%v]", raddr)
	response := Respond(query, zones, logHead)
	_, err := conn.WriteToUDP(response, raddr)
	if err != nil {
		log.Errorf("Can't write to socket: %v", err)
	}
}

func (h *SocketList) Set(s string) error {
	addr, err := net.ResolveUDPAddr("udp", s)
	if err != nil {
		return err
	}
	*h = append(*h, *addr)
	return nil
}

func (h *SocketList) String() string {
	return fmt.Sprint(*h)
}
