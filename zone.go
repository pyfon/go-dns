package main

import (
	"io"
	"net/netip"
)

type RecType int

const (
	TypeA RecType = iota
	TypeAAAA
	TypeCNAME
	TypeTXT
	TypeMX
	TypeNS
)

type Record struct {
	Name string
	Type RecType
	Addr netip.Addr // A, AAAA
	Target string // CNAME NS, MX...
	TXT []string // TXT
	TTL	uint // Seconds
}

type Zone struct {
	Zone string
	Records []Record
}

// parseZone returns a
func parseZone(reader io.Reader) (Zone, error) {
	return Zone{}, nil
}
