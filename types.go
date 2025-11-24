package main

import (
	"fmt"
	"net/netip"
	"strings"
	"regexp"
)

type Domain string
type RecordName string // E.g. wow.example, *.example for zone "com."
type RecType string
type TXTData [][]byte

const (
	TypeA     RecType = "A"
	TypeAAAA  RecType = "AAAA"
	TypeCNAME RecType = "CNAME"
	TypeTXT   RecType = "TXT"
	TypeMX    RecType = "MX"
	TypeNS    RecType = "NS"
)

type Record struct {
	Name   RecordName
	Type   RecType
	Addr   netip.Addr // A, AAAA
	Target Domain     // For CNAMEs, MX etc
	TXT    TXTData    // TXT, split into 255-byte strings.
	TTL    uint       // Seconds
}

type Zone struct {
	Name    Domain            // Domain the zone is responsible for.
	TTL     uint              // Default TTL in seconds
	Records map[string]Record // Map of records indexed by name
}

// domainRegex defines a regex for a valid domain name. This does NOT include @ and wildcard domains.
var domainRegex *regexp.Regexp = regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.?)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9]\.?$`)

// Matches valid record names like "example" and "*.example", "@"
var recordNameRegex *regexp.Regexp = regexp.MustCompile(`^(?:@|\*|(?:\*\.)?(?:[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?)(?:\.(?:[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?))*)$`)

func NewTXTData(data string) TXTData {
	b := []byte(data)
    var out [][]byte

    for len(b) > 255 {
        out = append(out, b[:255])
        b = b[255:]
    }
    out = append(out, b)

    return TXTData(out)
}

func NewZone() Zone {
	return Zone{
		Records: make(map[string]Record),
	}
}

// FQDN reports whether Domain is fully-qualified. It does not check for domain validity.
func (d Domain) FQDN() bool {
	return d[len(d)-1:] == "."
}

// Parent returns the domains parent. For example the parent of a.example.com is "example.com".
// When the domain is already a TLD/root, the original domain will be returned and tld will be true.
func (d Domain) Parent() (domain Domain, tld bool) {
	segments := strings.Split(string(d), ".")
	if len(segments) <= 1 {
		return d, true
	}
	return Domain(strings.Join(segments[1:], ".")), false
}

func (d Domain) String() string {
	return string(d)
}

// Valid reports whether the domain is a valid domain name.
func (d Domain) Valid() bool {
	return domainRegex.MatchString(d.String())
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
		return r.TXT.String()
	}
	return ""
}

func (r RecordName) Root() bool {
	return r == "@"
}

func (r RecordName) String() string {
	return string(r)
}

// Valid reports whether the domain is a valid record name.
func (r RecordName) Valid() bool {
	return recordNameRegex.MatchString(r.String())
}

func (t TXTData) String() string {
	var builder strings.Builder
	for _, s := range t {
		builder.WriteString(string(s))
	}
	return builder.String()
}

func (z Zone) String() string {
	var s strings.Builder
	s.WriteString("----------\n")
	s.WriteString(fmt.Sprintf("ZONE %v\nTTL: %v\nRecords:\n", z.Name, z.TTL))
	for _, r := range z.Records {
		// Ideally, this needs printing in a proper tabular format.
		rStr := fmt.Sprintf("%v\t%v\t%v\t\t\t%v\n", r.Name, r.Type, r.dataString(), r.TTLOrDefault(z))
		s.WriteString(rStr)
	}
	s.WriteString("----------\n")
	return s.String()
}
