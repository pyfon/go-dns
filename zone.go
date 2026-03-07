package main

import (
	"errors"
	"fmt"
	"net/netip"
	"regexp"
	"strings"
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
	Records Trie[RData]
}

type RData struct {
	Initialised bool
	empty bool
	hasCNAME bool
	rdata map[RecType][]Record
}

// domainRegex defines a regex for a valid domain name. This does NOT include @ and wildcard domains.
var domainRegex *regexp.Regexp = regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.?)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9]\.?$`)

// Matches valid record names like "example" and "*.example", "@"
var recordNameRegex *regexp.Regexp = regexp.MustCompile(`^(?:@|\*|(?:\*\.)?(?:[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?)(?:\.(?:[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?))*)$`)

func NewZone() Zone {
	return Zone{
		Records: NewTrie[RData](),
	}
}

func NewZoneTrie(zones map[Domain]Zone) Trie[Zone] {
	trie := NewTrie[Zone]()
	for _, zone := range zones {
		trie.Insert(string(zone.Name), zone)
	}
	return trie
}

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

func NewRData() RData {
	return RData{
		Initialised: true,
		empty: true,
		hasCNAME: false,
		rdata: make(map[RecType][]Record),
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

// TrieKey converts r into a key for a Trie[RData].
// Wildcards simply have a node referenced by "*".
func (r RecordName) TrieKey(zoneName Domain) string {
	if r.Root() {
		return "" // An empty key yields the root node.
	}
	return r.String()
}

// Valid reports whether the domain is a valid record name.
func (r RecordName) Valid() bool {
	return recordNameRegex.MatchString(r.String())
}

// Insert will add the given record to RDATA.
func (r *RData) Insert(record Record) error {
	recIsCNAME := record.Type == TypeCNAME
	if r.hasCNAME {
		errStr := fmt.Sprintf("%v is a CNAME and cannot have any other records", record.Name)
		return errors.New(errStr)
	}
	if recIsCNAME && !r.empty {
		errStr := fmt.Sprintf("Cannot add CNAME %v, other records cannot exist beside a CNAME", record.Name)
		return errors.New(errStr)
	}
	r.rdata[record.Type] = append((*r).rdata[record.Type], record)
	r.empty = false
	r.hasCNAME = recIsCNAME
	return nil
}

func (t TXTData) String() string {
	var builder strings.Builder
	for _, s := range t {
		builder.WriteString(string(s))
	}
	return builder.String()
}
