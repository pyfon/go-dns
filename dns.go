package main

import (
	"fmt"
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
		return errReply(query, rcodeFormErr, logHead)
	}
	// For logging purposes:
	qInfo := queryInfo(query)

	answers := make(map[Domain]RData)
	for _, q := range query.Question {
		zone, _ := zones.Search(q.Name.AsFQDN().String())
		rrset, found, err := zone.Query(queryStr(zone, q.Name))
		if err != nil {
			log.Errorf("%v Error when querying zone for query %v, returning SERVFAIL: %v", logHead, qInfo, err)
			return errReply(query, rcodeServFail, logHead)
		}
		if !found {
			log.Infof("%v [NXDOMAIN] %v", logHead, qInfo)
			return errReply(query, rcodeNxdomain, logHead)
		}
		// Could we improve this to make it recursive? :
		ty := q.Type
		if rrset.HasCNAME {
			ty = TypeCNAME
		}
		for rdata := range rrset.Get(ty) {
			answers[Domain(q.Name)] = rdata
		}
	}

	replyMsg, err := NewDNSMsg(query, answers)
	if err != nil {
		log.Errorf("%v Error when constructing NewDNSMsg for reply to %v: %v", logHead, qInfo, err)
		return errReply(query, rcodeServFail, logHead)
	}

	reply, err := replyMsg.Serialise()
	if err != nil {
		log.Errorf("%v Could not serialise reply to %v: %v", logHead, qInfo, err)
		return errReply(query, rcodeServFail, logHead)
	}

	log.Infof("%v [NoError] %v", logHead, qInfo)
	return reply
}

// errReply constructs a serialised error response.
func errReply(orig DNSMsg, rcode byte, logHead string) []byte {
	reply := NewDNSMsgErr(orig, rcode)
	payload, err := reply.Serialise()
	if err != nil {
		log.Errorf("%v BUG? Error when serialising error reply: %v. Replying with NULL", logHead, err)
		return []byte("")
	}
	return payload
}

// queryInfo returns a string which describes a DNS query, for logging purposes.
func queryInfo(msg DNSMsg) string {
	var s string
	sep := ""
	for _, q := range msg.Question {
		s = fmt.Sprintf("%vName: %v Type: %v%v", s, q.Name, q.Type.String(), sep)
		sep = " | "
	}
	return s
}

// queryStr returns the appropriate query key to use to search for the (absolute) name given in zone.
func queryStr(zone *Zone, name Domain) Domain {
	nameFQDN := name.AsFQDN()
	if nameFQDN == zone.Name {
		return ""
	}
	key, _ := strings.CutSuffix(nameFQDN.String(), "."+zone.Name.String())
	return Domain(key)
}
