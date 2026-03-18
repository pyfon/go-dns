package main

import (
	log "github.com/sirupsen/logrus"
)

// Respond will respond to a DNS query using the given zones.
// query is the full query from the wire, truncated to the request data (no zeroes from the buffer).
// logHead is a string containing information about the request for logging purposes.
func Respond(query []byte, zones *Trie[Zone], logHead string) []byte {
	err := LogQuestion(query)
	if err != nil {
		log.Error(err)
		return []byte("Uh oh")
	}
	return []byte("Hello")
}
