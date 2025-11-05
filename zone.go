package main

import (
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
	Name   string
	Type   RecType
	Addr   netip.Addr // A, AAAA
	Target Domain     // For CNAMEs, MX etc
	TXT    []string   // TXT
	TTL    uint       // Seconds
}

type Zone struct {
	Zone    Domain
	TTL     int               // Default TTL in seconds
	Records map[string]Record // Map of records indexed by name
}
