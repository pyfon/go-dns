package main

import (
	"context"
	"fmt"
	"net"
	"time"

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

	go func() {
		<-ctx.Done()
		conn.SetReadDeadline(time.Now())
	}()

	for {
		buf := make([]byte, 512)
		n, raddr, err := conn.ReadFromUDP(buf)
		if err := ctx.Err(); err != nil { // TODO add a timeout context to ReadFromUDP?
			log.Errorf("Shutting down listener for socket %v: %v", sock, err)
			return err
		}
		if err != nil {
			log.Errorf("Could not read UDP request: %v", err)
			continue
		}
		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		go Handle(conn, raddr, buf[:n], &zones, timeoutCtx)
	}
}

// Handle will handle a connection.
func Handle(conn *net.UDPConn, raddr *net.UDPAddr, query []byte, zones *Trie[Zone], ctx context.Context) {
	logHead := fmt.Sprintf("[%v]", raddr)
	response := Respond(query, zones, logHead)
	_, err := conn.WriteToUDP(response, raddr)
	if err := ctx.Err(); err != nil {
		log.Errorf("Stopped writing to UDP socket due to context: %v", err)
	}
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
