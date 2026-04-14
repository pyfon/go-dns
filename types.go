package main

import (
	"fmt"
	"net/netip"
	"regexp"
	"strings"
)

// These RecType values correspond to the DNS message values for the given type.
// To add another type, just add one of these const values and an entry to recTypeToName.
const (
	TypeA     RecType = 1
	TypeNS    RecType = 2
	TypeCNAME RecType = 5
	TypePTR   RecType = 12
	TypeMX    RecType = 15
	TypeTXT   RecType = 16
	TypeAAAA  RecType = 28
	TypeOPT   RecType = 41
)

const (
	QClassIN QClass = 1
)

type Domain string
type RecType uint16
type QClass uint16
type TXTData [][]byte

type RData struct {
	Name   RecordName
	Type   RecType
	Addr   netip.Addr // A, AAAA
	Target Domain     // For CNAMEs, MX etc. The zonefile parser always makes the target an FQDN.
	TXT    TXTData    // TXT, split into 255-byte strings
	TTL    uint       // Seconds
	Pref   uint16     // For MX
}

// domainRegex defines a regex for a valid domain name. This does NOT include @ and wildcard domains.
var domainRegex *regexp.Regexp = regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.?)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9]\.?$`)

var recTypeToName = map[RecType]string{
	TypeA:     "A",
	TypeNS:    "NS",
	TypeCNAME: "CNAME",
	TypePTR:   "PTR",
	TypeMX:    "MX",
	TypeTXT:   "TXT",
	TypeAAAA:  "AAAA",
	TypeOPT:   "OPT",
}

var qClassByName = map[string]QClass{
	"IN": QClassIN,
}

// NewTXTData converts a string of arbitrary length to TXTData.
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

// AsFQDN converts d into an FQDN (adds a "." suffix if not present)
func (d Domain) AsFQDN() Domain {
	if d.FQDN() {
		return d
	}
	return d + "."
}

// FQDN reports whether Domain is fully-qualified. It does not check for domain validity.
func (d Domain) FQDN() bool {
	l := len(d)
	if l == 0 {
		return false
	}
	return d[l-1:] == "."
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

// Labels splits the domain into a slice of labels.
func (d Domain) Labels() []string {
	if d.FQDN() {
		s, _ := strings.CutSuffix(d.String(), ".")
		d = Domain(s)
	}
	return strings.Split(d.String(), ".")
}

func (d Domain) String() string {
	return string(d)
}

// Valid reports whether the domain is a valid domain name.
func (d Domain) Valid() bool {
	return domainRegex.MatchString(d.String())
}

func (q QClass) Valid() bool {
	if q == QClassIN {
		return true
	}
	return false
}

func (q QClass) String() string {
	if q == QClassIN {
		return "IN"
	}
	return ""
}

func (t TXTData) String() string {
	var builder strings.Builder
	for _, s := range t {
		builder.WriteString(string(s))
	}
	return builder.String()
}

// TTLOrDefault returns the TTL of the record, falling back to the default of zone if new TTL was specified
func (r RData) TTLOrDefault(zone Zone) uint {
	if r.TTL == 0 {
		return zone.TTL
	}
	return r.TTL
}

// dataString returns a string representation of the data/target/txt depending on the record type.
func (r RData) DataString() string {
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

func (r RecType) Valid() bool {
	for k, _ := range recTypeToName {
		if k == r {
			return true
		}
	}
	return false
}

func (r RecType) String() string {
	if s, ok := recTypeToName[r]; ok {
		return s
	}
	return ""
}

// ParseQClass converts a string to a QClass.
func ParseQClass(s string) (QClass, error) {
	if t, ok := qClassByName[s]; ok {
		return t, nil
	}
	return 0, fmt.Errorf("Unknown QCLASS: %q", s)
}

// ParseRecType converts a string to a RecType.
func ParseRecType(s string) (RecType, error) {
	for k, v := range recTypeToName {
		if s == v {
			return k, nil
		}
	}
	return 0, fmt.Errorf("Unknown record type: %q", s)
}
