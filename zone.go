package main

import (
	"bufio"
	"errors"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

type RecordName string // E.g. wow.example, *.example for zone "com."
type TXTData [][]byte

type Zone struct {
	Name    Domain // Domain the zone is responsible for.
	TTL     uint   // Default TTL in seconds
	Records map[string]RRSet
}

type RRSet struct {
	Empty    bool
	HasCNAME bool
	RRSet    map[RecType][]RData
}

// Matches valid record names like "example" and "*.example", "@"
var recordNameRegex *regexp.Regexp = regexp.MustCompile(`^(?:@|\*|(?:\*\.)?(?:[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?)(?:\.(?:[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?))*)$`)

func NewZone() Zone {
	return Zone{
		Records: make(map[string]RRSet),
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

func NewRRSet() RRSet {
	return RRSet{
		Empty:    true,
		HasCNAME: false,
		RRSet:    make(map[RecType][]RData),
	}
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
func (r *RRSet) Get(t RecType) iter.Seq[RData] {
	return func(yield func(RData) bool) {
		for _, v := range r.RRSet[t] {
			if !yield(v) {
				return
			}
		}
	}
}

// GetAll is an iterator which will return all records in r, one at a time.
func (r *RRSet) GetAll() iter.Seq[RData] {
	return func(yield func(RData) bool) {
		for _, v := range r.RRSet {
			for _, rec := range v {
				if !yield(rec) {
					return
				}
			}
		}
	}
}

// CNAME will yield the CNAME RDATA of r, as long as r is a CNAME.
func (r *RRSet) CNAME() RData {
	if !r.HasCNAME {
		log.Errorf("GetCNAME() called when RRSet is not a CNAME")
		return RData{}
	}
	return r.RRSet[TypeCNAME][0]
}

// Insert will add the given record to RRSet.
func (r *RRSet) Insert(record RData) error {
	recIsCNAME := record.Type == TypeCNAME
	if r.HasCNAME {
		errStr := fmt.Sprintf("%v is a CNAME and cannot have any other records", record.Name)
		return errors.New(errStr)
	}
	if recIsCNAME && !r.Empty {
		errStr := fmt.Sprintf("Cannot add CNAME %v, other records cannot exist beside a CNAME", record.Name)
		return errors.New(errStr)
	}
	r.RRSet[record.Type] = append(r.RRSet[record.Type], record)
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

// Query will return a RRSet for the given name. Name is taken to be the subdomain within the zone.
// E.g. "x" for x.example.com in zone example.com. "" is taken to mean the zone root.
// If an exact match isn't found, a wildcard lookup will be attempted and returned if successful.
// If no match can be found, the returned bool will be false. If a match is returned the bool will be true.
func (z *Zone) Query(name Domain) (RRSet, bool, error) {
	if name.FQDN() {
		return RRSet{}, false, errors.New("Queried name cannot be an FQDN.")
	}

	nameStr := name.String()
	RRSet, ok := z.Records[nameStr]
	if ok {
		return RRSet, true, nil
	}

	// No exact match, try a wildcard match by replacing the leftmost label with *
	_, after, _ := strings.Cut(nameStr, ".")
	sep := "."
	if after == "" {
		sep = ""
	}
	nameStr = "*" + sep + after

	RRSet, ok = z.Records[nameStr]
	return RRSet, ok, nil
}

// Insert will insert the record into the zone.
func (z *Zone) Insert(record RData) error {
	recName := record.Name.String()
	if record.Name.Root() {
		recName = "" // An empty key yields the root node.
	}
	val, ok := z.Records[recName]
	if !ok {
		val = NewRRSet()
	}
	val.Insert(record)
	z.Records[recName] = val
	return nil
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
// Please note the returned Trie should be searched via FQDN, as the root zone is "".
func ParseZoneFiles(zoneDirPath string) (Trie[Zone], error) {
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
