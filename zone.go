package main

import (
	"errors"
	"fmt"
	"net/netip"
	"strings"
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
	Name   Domain // Does not include zone part, not an FQDN.
	Type   RecType
	Addr   netip.Addr // A, AAAA
	Target Domain     // For CNAMEs, MX etc
	TXT    string     // TXT
	TTL    uint       // Seconds
}

type Zone struct {
	Zone    Domain
	TTL     uint              // Default TTL in seconds
	Records map[string]Record // Map of records indexed by name
}

func newZone() Zone {
	return Zone{
		Records: make(map[string]Record),
	}
}

func ParseRecType(s string) (RecType, error) {
	switch s {
	case "A":
		return TypeA, nil
	case "AAAA":
		return TypeAAAA, nil
	case "CNAME":
		return TypeCNAME, nil
	case "TXT":
		return TypeTXT, nil
	case "MX":
		return TypeMX, nil
	case "NS":
		return TypeNS, nil
	}
	errStr := fmt.Sprintf("Unknown record type: %v", s)
	return 0, errors.New(errStr)
}

func (z Zone) String() string {
	var s strings.Builder
	s.WriteString("----------\n")
	s.WriteString(fmt.Sprintf("ZONE %v\nTTL: %v\nRecords:\n", z.Zone, z.TTL))
	for _, r := range z.Records {
		// Ideally, this needs printing in a proper tabular format.
		rStr := fmt.Sprintf("%v\t%v\t%v\t\t\t%v\n", r.Name, r.Type.String(), r.dataString(), r.TTLOrDefault(z))
		s.WriteString(rStr)
	}
	s.WriteString("----------\n")
	return s.String()
}

// TTLOrDefault returns the TTL of the record, falling back to the default of zone if new TTL was specified
func (r Record) TTLOrDefault(zone Zone) uint {
	if r.TTL == 0 {
		return zone.TTL
	}
	return r.TTL
}

// dataString returns a string representation of the data/target/txt depending on the record type.
func (r Record) dataString() string {
	switch r.Type {
	case TypeA, TypeAAAA:
		return r.Addr.String()
	case TypeCNAME, TypeMX, TypeNS:
		return r.Target.String()
	case TypeTXT:
		return r.TXT
	}
	return ""
}

func (r RecType) String() string {
	switch r {
	case TypeA:
		return "A"
	case TypeAAAA:
		return "AAAA"
	case TypeCNAME:
		return "CNAME"
	case TypeTXT:
		return "TXT"
	case TypeMX:
		return "MX"
	case TypeNS:
		return "NS"
	}
	return ""
}
