package main

import (
	"fmt"
	"regexp"
	"strings"
)

type Domain string
type RecType uint16
type QClass uint16

// These RecType values correspond to the DNS message values for the given type.
const (
	TypeA     RecType = 1
	TypeNS    RecType = 2
	TypeCNAME RecType = 5
	TypeSOA   RecType = 6
	TypePTR   RecType = 12
	TypeMX    RecType = 15
	TypeTXT   RecType = 16
	TypeAAAA  RecType = 28
)

const (
	QClassIN QClass = 1
)

// domainRegex defines a regex for a valid domain name. This does NOT include @ and wildcard domains.
var domainRegex *regexp.Regexp = regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.?)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9]\.?$`)

var recTypeByName = map[string]RecType{
	"A":     TypeA,
	"NS":    TypeNS,
	"CNAME": TypeCNAME,
	"SOA":   TypeSOA,
	"PTR":   TypePTR,
	"MX":    TypeMX,
	"TXT":   TypeTXT,
	"AAAA":  TypeAAAA,
}

var qClassByName = map[string]QClass{
	"IN": QClassIN,
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

func (r RecType) Valid() bool {
	switch r {
	case TypeA, TypeNS, TypeCNAME, TypeSOA, TypePTR, TypeMX, TypeTXT, TypeAAAA:
		return true
	}
	return false
}

func (r RecType) String() string {
	switch r {
	case TypeA:
		return "A"
	case TypeNS:
		return "NS"
	case TypeCNAME:
		return "CNAME"
	case TypeSOA:
		return "SOA"
	case TypePTR:
		return "PTR"
	case TypeMX:
		return "MX"
	case TypeTXT:
		return "TXT"
	case TypeAAAA:
		return "AAAA"
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
	if t, ok := recTypeByName[s]; ok {
		return t, nil
	}
	return 0, fmt.Errorf("Unknown record type: %q", s)
}
