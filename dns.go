package main

import (
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
	if len(qSplit) < 2 {
		return []byte("ERROR: Invalid query. Format: <domain> <type>\n")
	}
	domain := Domain(strings.TrimSpace(qSplit[0]))
	queryType := strings.TrimSpace(qSplit[1])

	if !domain.Valid() {
		return []byte("ERROR: Invalid domain\n")
	}
	if !domain.FQDN() {
		// All queries are considered full-qualified
		domain = domain + "."
	}

	zone, _ := zones.Search(domain.String())
	log.Debugf("%v Got zone %v", logHead, zone.Name)

	var search string // The record within the zone to search for.
	if domain == zone.Name {
		search = ""
	} else {
		search = strings.TrimSuffix(domain.String(), "."+zone.Name.String())
	}

	// TODO Rewrite all this horrible shit.
	var rdata *RData
	for {
		var exists bool
		rdata, exists = zone.Records.Search(search)
		if exists {
			break
		}
		before, _, _ := strings.Cut(search, ".")
		if before == "*" {
			return []byte("ERROR: No results\n")
		}
		log.Debugf("RDATA doesn't exist for domain %v in rdata tree, trying wildcard", domain)
		_, after, _ := strings.Cut(search, ".")
		search = "*." + after
	}

	records := rdata.Get(RecType(queryType))
	if len(records) < 1 {
		return []byte("ERROR: No results\n")
	}

	var response []byte

	for _, record := range records {
		response = append(response, []byte(record.DataString()+"\n")...)
	}
	return response
	// --- TODO REMOVE ---
}
