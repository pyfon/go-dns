package main

import (
	"encoding/binary"
	"errors"
	"net/netip"

	log "github.com/sirupsen/logrus"
)

// iota isn't used here for clarity regarding the DNS protocol.
const (
	// Header QR flags
	qrQuery byte = 0
	qrReply byte = 1
	// Header opcode flags
	opcodeQuery  byte = 0
	opcodeIquery byte = 1
	opcodeStatus byte = 2
	opcodeMax    byte = 3 // Invalid opcodes start here.
	// Header response code flags
	rcodeNoError        byte = 0
	rcodeFormErr        byte = 1
	rcodeServFail       byte = 2
	rcodeNxdomain       byte = 3
	rcodeNotImplemented byte = 4
	rcodeRefused        byte = 5
	rcodeMax            byte = 6 // Invalid rcodes start here.
)

type Header struct {
	ID      uint16
	QR      byte // Query or reply?
	Opcode  byte
	AA      bool // Authoritative Answer
	TC      bool // Truncated message?
	RD      bool // Recursion Desired
	RA      bool // Recursion Available
	AD      bool // Authentic Data
	CD      bool // Checking disabled
	Rcode   byte
	QDCount uint16 // Number of questions
	ANCount uint16 // Number of answers
	NSCount uint16 // Number of Authority RRs
	ARCount uint16 // Number of Additional RRs
}

type Question struct {
	Name  Domain
	Type  RecType
	Class QClass
}

type RR struct {
	Name  Domain
	Type  RecType
	Class QClass
	TTL   uint32
	RData RData
}

// parseName will parse the "name" section or a domain of a DNS message segment.
// d is the parsed domain, offset is the total length of the name section,
// i.e. buf[offset] is the start of the next section.
func parseName(buf []byte) (d Domain, offset uint, err error) {
	sep := ""
	for {
		octets := uint(buf[offset])
		offset++
		if octets == 0 { // NULL, end of QNAME.
			if offset <= 1 {
				err = errors.New("Invalid question: no QNAME (first byte NULL)")
				return
			}
			break
		}
		if uint(len(buf)) <= offset+octets {
			err = errors.New("Name buffer is too small for domain")
			return
		}
		label := string(buf[offset : offset+octets])
		d += Domain(sep + label)
		sep = "."
		offset += octets
	}

	return
}

func parseHeader(buf [12]byte) (Header, error) {
	var header Header

	header.ID = binary.BigEndian.Uint16(buf[:2])

	flags := binary.BigEndian.Uint16(buf[2:4])
	header.QR = byte((flags >> 15) & 1)
	header.Opcode = byte((flags >> 11) & 0xF)
	header.AA = flags&(1<<10) != 0
	header.TC = flags&(1<<9) != 0
	header.RD = flags&(1<<8) != 0
	header.RA = flags&(1<<7) != 0
	// --- 1 reserved zero bit here ---
	header.AD = flags&(1<<5) != 0
	header.CD = flags&(1<<4) != 0
	header.Rcode = byte(flags & 0xF)

	header.QDCount = binary.BigEndian.Uint16(buf[4:6])
	header.ANCount = binary.BigEndian.Uint16(buf[6:8])
	header.NSCount = binary.BigEndian.Uint16(buf[8:10])
	header.ARCount = binary.BigEndian.Uint16(buf[10:12])

	if header.Opcode >= opcodeMax || header.Rcode >= rcodeMax {
		err := errors.New("Invalid OPCODE or RCODE")
		return header, err
	}

	return header, nil
}

// parseQuestion decodes one question from a question section of a DNS message from the start of buf.
// offset is the length of the parsed question read from buf in bytes.
func parseQuestion(buf []byte) (question Question, offset uint, err error) {
	err_small := errors.New("Question buffer is too small")
	if len(buf) <= 0 {
		err = err_small
		return
	}

	question.Name, offset, err = parseName(buf)
	if err != nil {
		return
	}

	if uint(len(buf)) < offset+4 { // +4 for QTYPE + QCLASS
		err = err_small
		return
	}

	question.Type = RecType(binary.BigEndian.Uint16(buf[offset : offset+2]))
	offset += 2
	question.Class = QClass(binary.BigEndian.Uint16(buf[offset : offset+2]))
	offset += 2

	if !question.Name.Valid() {
		err = errors.New("Invalid Domain in question section")
	}
	if !question.Type.Valid() {
		err = errors.New("Invalid or unsupported record type in question section")
	}
	if !question.Class.Valid() {
		err = errors.New("Invalid or unsupported class in question section")
	}

	return
}

func parseRR(buf []byte) (rr RR, offset uint, err error) {
	err_small := errors.New("RR buffer is too small")
	if len(buf) <= 0 {
		err = err_small
		return
	}

	rr.Name, offset, err = parseName(buf)
	if err != nil {
		return
	}

	if uint(len(buf)) <= offset+10 {
		err = err_small
		return
	}

	rr.Type = RecType(binary.BigEndian.Uint16(buf[offset : offset+2]))
	offset += 2
	rr.Class = QClass(binary.BigEndian.Uint16(buf[offset : offset+2]))
	offset += 2
	rr.TTL = binary.BigEndian.Uint32(buf[offset : offset+4])
	offset += 4

	rdLen := binary.BigEndian.Uint16(buf[offset : offset+2])
	offset += 2

	if uint(len(buf)) <= offset+uint(rdLen) {
		err = err_small
		return
	}

	rr.RData, err = parseRData(rr.Type, rr.TTL, buf[offset:offset+uint(rdLen)])

	return
}

func parseRData(t RecType, ttl uint32, buf []byte) (rdata RData, err error) {
	rdata.Type = t
	rdata.TTL = uint(ttl)

	switch t {
	case TypeA:
		rdata.Addr = netip.AddrFrom4([4]byte(buf))
	case TypeNS, TypeCNAME, TypePTR:
		rdata.Target, _, err = parseName(buf)
	case TypeMX:
		rdata.Pref = binary.BigEndian.Uint16(buf)
		rdata.Target, _, err = parseName(buf[2:])
	case TypeTXT:
		rdata.TXT = NewTXTData(string(buf))
	case TypeAAAA:
		rdata.Addr = netip.AddrFrom16([16]byte(buf))
	default:
		err = errors.New("Unknown RDATA type")
	}

	if err != nil {
		return
	}
	if (t == TypeA || t == TypeAAAA) && !rdata.Addr.IsValid() {
		err = errors.New("Invalid A/AAAA address in RDATA")
		return
	}

	return
}

func boolToUint16(b bool) uint16 {
	if b {
		return 1
	}
	return 0
}

// Serialise will convert h into the header of a DNS message.
func (h Header) Serialise() (payload []byte) {
	payload = binary.BigEndian.AppendUint16(payload, h.ID)

	var flags uint16
	flags |= uint16(h.QR) << 15
	flags |= uint16(h.Opcode) << 11
	flags |= boolToUint16(h.AA) << 10
	flags |= boolToUint16(h.TC) << 9
	flags |= boolToUint16(h.RD) << 8
	flags |= boolToUint16(h.RA) << 7
	// --- Reserved zero bit here. ---
	flags |= boolToUint16(h.AD) << 5
	flags |= boolToUint16(h.CD) << 4
	flags |= uint16(h.Rcode)
	payload = binary.BigEndian.AppendUint16(payload, flags)

	payload = binary.BigEndian.AppendUint16(payload, h.QDCount)
	payload = binary.BigEndian.AppendUint16(payload, h.ANCount)
	payload = binary.BigEndian.AppendUint16(payload, h.NSCount)
	payload = binary.BigEndian.AppendUint16(payload, h.ARCount)

	return payload
}

// Serialise will convert q into a single question of a question section of a DNS message.
func (q Question) Serialise() (payload []byte) {
	for _, l := range q.Name.Labels() {
		payload = append(payload, byte(len(l)))
		payload = append(payload, []byte(l)...)
	}
	payload = append(payload, 0)

	payload = binary.BigEndian.AppendUint16(payload, uint16(q.Type))
	payload = binary.BigEndian.AppendUint16(payload, uint16(q.Class))
	return
}

// TODO REMOVE THIS! For dev purposes.
func LogQuestion(buf []byte) error {
	if len(buf) < 12 {
		return errors.New("Request too small")
	}
	header, err := parseHeader([12]byte(buf[:12]))
	if err != nil {
		return err
	}

	var questions []Question
	for i := 0; i < int(header.QDCount); i++ {
		question, _, err := parseQuestion(buf[12:])
		if err != nil {
			return err
		}
		questions = append(questions, question)
	}

	for _, question := range questions {
		log.Infof("Got question: %v %v", question.Name, question.Type)
	}
	return nil
}
