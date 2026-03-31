package main

import (
	"strings"

	log "github.com/sirupsen/logrus"
)

// Respond will respond to a DNS query using the given zones.
// query is the full query from the wire, truncated to the request data (no zeroes from the buffer).
// logHead is a string containing information about the request for logging purposes.
func Respond(queryBuf []byte, zones *Trie[Zone], logHead string) []byte {
	query, err := ParseDNSMsg(queryBuf)
	if err != nil {
		log.Errorf("%v Error when parsing request: %v", logHead, err)
		return errReply(query, rcodeFormErr)
	}

	var answers map[Domain]RData
	log.Debugf("%v Query contains %v questions", logHead, len(query.Question))
	for _, q := range query.Question {
		zone, _ := zones.Search(q.Name.String())
		subdomain, _ := strings.CutSuffix(q.Name.String(), "."+zone.Name.String())
		log.Debugf("%v Querying zone %v for record %v", logHead, zone.Name, subdomain)
		rrset, found, err := zone.Query(Domain(subdomain))
		if err != nil {
			log.Errorf("%v Error when querying zone, returning SERVFAIL: %v", logHead, err)
			return errReply(query, rcodeServFail)
		}
		if !found {
			log.Infof("%v Could not find answer to query %v, returning NXDOMAIN", logHead, q.Name)
			return errReply(query, rcodeNxdomain)
		}
		for rdata := range rrset.Get(q.Type) {
			answers[Domain(rdata.Name)] = rdata
		}
	}

	replyMsg, err := NewDNSMsg(query, answers)
	if err != nil {
		log.Errorf("%v Error when constructing NewDNSMsg for reply: %v", logHead, err)
		return errReply(query, rcodeServFail)
	}

	reply, err := replyMsg.Serialise()
	if err != nil {
		log.Errorf("%v Could not serialise reply: %v", logHead, err)
		return errReply(query, rcodeServFail)
	}

	return reply
}

// errReply constructs a serialised error response.
func errReply(orig DNSMsg, rcode byte) []byte {
	reply := NewDNSMsgErr(orig, rcode)
	payload, err := reply.Serialise()
	if err != nil {
		log.Errorf("%v BUG? Error when serialising error reply: %v. Replying with NULL", err)
		return []byte("")
	}
	return payload
}
