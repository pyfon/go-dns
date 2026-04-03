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

	logHead = fmt.Sprintf("%s [%s]", logHead, queryInfo(query))

	answers := make(map[Domain][]RData)
	for _, q := range query.Question {
		a, ok, errMsg := answer(q, zones, query, logHead)
		if !ok {
			return errMsg
		}
		answers[q.Name] = append(answers[q.Name], a...)
	}

	replyMsg, err := NewDNSMsg(query, answers)
	if err != nil {
		log.Errorf("%v Error when constructing NewDNSMsg for reply: %v", logHead, err)
		return errReply(query, rcodeServFail, logHead)
	}

	reply, err := replyMsg.Serialise()
	if err != nil {
		log.Errorf("%v Could not serialise reply: %v", logHead, err)
		return errReply(query, rcodeServFail, logHead)
	}

	log.Infof("%v [NoError]", logHead)
	return reply
}

// answer attempts to recursively answer one question using all the given zones.
// If an answer was found, ok will be true and the answers will be returned.
// Else, ok will be false, and an error reply DNS message will be given.
func answer(q Question, zones *Trie[Zone], orig DNSMsg, logHead string) (answers []RData, ok bool, errMsg []byte) {
	zone, _ := zones.Search(q.Name.AsFQDN().String())
	rrset, found, err := zone.Query(queryStr(zone, q.Name))
	if err != nil {
		log.Errorf("%v Error when querying zone for query, returning SERVFAIL: %v", logHead, err)
		errMsg = errReply(orig, rcodeServFail, logHead)
		return
	}
	if !found {
		log.Infof("%v [NXDOMAIN]", logHead)
		errMsg = errReply(orig, rcodeNxdomain, logHead)
		return
	}

	// Recursively search for an answer if we got a CNAME where none was requested.
	if q.Type != TypeCNAME && rrset.HasCNAME {
		cname := rrset.CNAME()
		answers = append(answers, cname)

		cnameQ := Question{Name: cname.Target, Type: TypeCNAME, Class: QClassIN}
		ans, ok, errMsg := answer(cnameQ, zones, orig, logHead)
		if !ok {
			return answers, false, errMsg
		}
		answers = append(answers, ans...)
		return answers, ok, errMsg
	}

	for rdata := range rrset.Get(q.Type) {
		answers = append(answers, rdata)
	}

	ok = true
	return
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
