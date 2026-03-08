package main

import (
	"fmt"
	"iter"
	"strings"

	log "github.com/sirupsen/logrus"
)

// Respond will respond to a DNS query using the given zones.
// query is the full query from the wire, truncated to the request data (no zeroes from the buffer).
// logHead is a string containing information about the request for logging purposes.
func Respond(query []byte, zones *Trie[Zone], logHead string) []byte {
	// --- TODO REMOVE ---
	log.Infof("%v Received query", logHead)
	qSplit := strings.Split(string(query), " ")
	if len(qSplit) < 1 {
		return []byte("ERROR: Invalid query. Format: <domain> [type]\n")
	}
	domain := Domain(strings.TrimSpace(qSplit[0]))
	var queryType string
	if len(qSplit) > 1 {
		queryType = strings.TrimSpace(qSplit[1])
	}

	if !domain.Valid() {
		return []byte("ERROR: Invalid domain\n")
	}
	if !domain.FQDN() {
		// All queries are considered full-qualified
		domain = domain + "."
	}

	zone, _ := zones.Search(domain.String())

	var search string // The record within the zone to search for.
	if domain == zone.Name {
		search = ""
	} else {
		search = strings.TrimSuffix(domain.String(), "."+zone.Name.String())
	}

	rdata, err := zone.Query(Domain(search))
	if err != nil {
		msg := fmt.Sprintf("Error when querying zone: %v", err)
		return []byte(msg)
	}

	var response []byte
	var recIter func() iter.Seq[Record] = rdata.GetAll
	if len(queryType) > 0 {
		recIter = func() iter.Seq[Record] { return rdata.Get(RecType(queryType)) }
	}

	for record := range recIter() {
		response = append(response, []byte(string(record.Type)+" "+record.DataString()+"\n")...)
	}
	return response
	// --- TODO REMOVE ---
}
