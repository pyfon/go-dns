package main

import (
	"fmt"
	"net"

	log "github.com/sirupsen/logrus"
)

type SocketList []net.UDPAddr

// Serve DNS on the given socket until program termination.
func Serve(sock net.UDPAddr, zones ZoneTrie) error {
	conn, err := net.ListenUDP("udp", &sock)
	if err != nil {
		log.Errorf("Could not serve on socket %v: %v", sock, err)
		return err
	}
	defer conn.Close()
	log.Infof("Serving DNS on %v:%v", sock.IP, sock.Port)

	for {
		buf := make([]byte, 512)
		_, raddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Errorf("Could not read UDP request: %v", err)
			continue
		}
		go Respond(conn, raddr, buf, &zones)
	}
}

// Respond to a DNS query.
func Respond(conn *net.UDPConn, raddr *net.UDPAddr, query []byte, zones *ZoneTrie) {
	// --- TODO REMOVE ---
	log.Infof("Received query from %v", raddr)
	_, err := conn.WriteToUDP([]byte("boo!"), raddr)
	if err != nil {
		log.Errorf("Can't write to socket: %v", err)
	}
	// --- TODO REMOVE ---
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
