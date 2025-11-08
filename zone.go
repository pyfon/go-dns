package main

import (
	"fmt"
	"net/netip"
	"strings"
)

type RecType string

const (
	TypeA     RecType = "A"
	TypeAAAA  RecType = "AAAA"
	TypeCNAME RecType = "CNAME"
	TypeTXT   RecType = "TXT"
	TypeMX    RecType = "MX"
	TypeNS    RecType = "NS"
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

// FindBestZoneMatch finds the zone which is the most specific match for domain in the zone map
// and returns a pointer to it.
// For example a.b.example.com would first match the b.example.com zone if present, if not example.com, if not com.
// This function will return nil if no match is found in the zones map.
func FindBestZoneMatch(zones map[Domain]*Zone, domain Domain) *Zone {
	curDomain := domain
	for {
		zone, ok := zones[curDomain]
		if ok {
			return zone
		}
		var tld bool
		curDomain, tld = curDomain.Parent()
		if tld {
			return nil // We've hit the root with no matches.
		}
	}
}

func (z Zone) String() string {
	var s strings.Builder
	s.WriteString("----------\n")
	s.WriteString(fmt.Sprintf("ZONE %v\nTTL: %v\nRecords:\n", z.Zone, z.TTL))
	for _, r := range z.Records {
		// Ideally, this needs printing in a proper tabular format.
		rStr := fmt.Sprintf("%v\t%v\t%v\t\t\t%v\n", r.Name, r.Type, r.dataString(), r.TTLOrDefault(z))
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
