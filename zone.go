package main

import (
	"bufio"
	"errors"
	"fmt"
	"iter"
	"net/netip"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
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
	Name    Domain // Domain the zone is responsible for.
	TTL     uint   // Default TTL in seconds
	Records map[string]RData
}

type RData struct {
	Empty    bool
	HasCNAME bool
	rdata    map[RecType][]Record
}

// domainRegex defines a regex for a valid domain name. This does NOT include @ and wildcard domains.
var domainRegex *regexp.Regexp = regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.?)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9]\.?$`)

// Matches valid record names like "example" and "*.example", "@"
var recordNameRegex *regexp.Regexp = regexp.MustCompile(`^(?:@|\*|(?:\*\.)?(?:[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?)(?:\.(?:[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?))*)$`)

func NewZone() Zone {
	return Zone{
		Records: make(map[string]RData),
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
		Empty:    true,
		HasCNAME: false,
		rdata:    make(map[RecType][]Record),
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
func (r Record) DataString() string {
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

// Get will retreive a slice of Records of a given type
func (r *RData) Get(t RecType) iter.Seq[Record] {
	return func(yield func(Record) bool) {
		for _, v := range r.rdata[t] {
			if !yield(v) {
				return
			}
		}
	}
}

// GetAll is an iterator which will return all records in r, one at a time.
func (r *RData) GetAll() iter.Seq[Record] {
	return func(yield func(Record) bool) {
		for _, v := range r.rdata {
			for _, rec := range v {
				if !yield(rec) {
					return
				}
			}
		}
	}
}

// Insert will add the given record to RDATA.
func (r *RData) Insert(record Record) error {
	recIsCNAME := record.Type == TypeCNAME
	if r.HasCNAME {
		errStr := fmt.Sprintf("%v is a CNAME and cannot have any other records", record.Name)
		return errors.New(errStr)
	}
	if recIsCNAME && !r.Empty {
		errStr := fmt.Sprintf("Cannot add CNAME %v, other records cannot exist beside a CNAME", record.Name)
		return errors.New(errStr)
	}
	r.rdata[record.Type] = append(r.rdata[record.Type], record)
	r.Empty = false
	r.HasCNAME = recIsCNAME
	return nil
}

func (t TXTData) String() string {
	var builder strings.Builder
	for _, s := range t {
		builder.WriteString(string(s))
	}
	return builder.String()
}

// Query will return a RData for the given name. Name is taken to be the subdomain within the zone.
// E.g. "x" for x.example.com in zone example.com. "" is taken to mean the zone root.
// If an exact match isn't found, a wildcard lookup will be attempted and returned if successful.
// If no match can be found, the returned bool will be false. If a match is returned the bool will be true.
func (z *Zone) Query(name Domain) (RData, bool, error) {
	if name.FQDN() {
		return RData{}, false, errors.New("Queried name cannot be an FQDN.")
	}

	nameStr := name.String()
	rdata, ok := z.Records[nameStr]
	if ok {
		return rdata, true, nil
	}

	// No exact match, try a wildcard match by replacing the leftmost label with *
	_, after, _ := strings.Cut(nameStr, ".")
	sep := "."
	if after == "" {
		sep = ""
	}
	nameStr = "*" + sep + after

	rdata, ok = z.Records[nameStr]
	return rdata, ok, nil
}

// Insert will insert the record into the zone.
func (z *Zone) Insert(record Record) error {
	recName := record.Name.String()
	if record.Name.Root() {
		recName = "" // An empty key yields the root node.
	}
	val, ok := z.Records[recName]
	if !ok {
		val = NewRData()
	}
	val.Insert(record)
	z.Records[recName] = val
	return nil
}

// getFiles returns a list of valid, resolved file paths of all files recursively found under dirPath.
func getZoneFilePaths(zoneDirPath string) ([]string, error) {
	var files []string

	evalDirEnt := func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		realPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			return err
		}
		absPath, err := filepath.Abs(realPath)
		if err != nil {
			return err
		}
		// Make sure the filename is *.zone
		base := filepath.Base(absPath)
		split := strings.Split(base, ".")
		if len(split) < 2 {
			return nil
		}
		if split[len(split)-1] != "zone" {
			return nil
		}
		files = append(files, absPath)
		return nil
	}

	if err := filepath.WalkDir(zoneDirPath, evalDirEnt); err != nil {
		return nil, err
	}

	return files, nil
}

// parseZoneFiles takes a list of zone file paths, parses each one into a Zone object,
// and returns a trie of Zones for fast lookup.
func parseZoneFiles(zoneDirPath string) (Trie[Zone], error) {
	log.Debugf("Parsing zone files in %s", zoneDirPath)
	zoneFiles, err := getZoneFilePaths(zoneDirPath)
	if err != nil {
		s := fmt.Sprintf("Couldn't gather zone files in %v: %v", zoneDirPath, err)
		err := errors.New(s)
		return Trie[Zone]{}, err
	}

	var zones map[Domain]Zone = make(map[Domain]Zone)
	for _, file := range zoneFiles {
		zoneFile, err := os.Open(file)
		if err != nil {
			return NewTrie[Zone](), err
		}
		zoneReader := bufio.NewReader(zoneFile)
		lexer := NewLexer(zoneReader)
		parser := NewParser(&lexer, filepath.Base(file))
		zone, err := parser.Parse()
		zoneFile.Close()
		if err != nil {
			return NewTrie[Zone](), err
		}
		if _, exists := zones[zone.Name]; exists {
			errStr := fmt.Sprintf("Duplicate zone: %v", zone.Name)
			return NewTrie[Zone](), errors.New(errStr)
		}
		zones[zone.Name] = zone
	}

	return NewZoneTrie(zones), nil
}
