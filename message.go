package main

import (
	"encoding/binary"
)

const (
	// Header QR flags
	qrQuery byte = 0
	qrReply byte = 1
	// Header opcode flags
	opcodeQuery  byte = 0
	opcodeIquery byte = 1
	opcodeStatus byte = 2
	// Header response code flags
	rcodeNoError        byte = 0
	rcodeFormErr        byte = 1
	rcodeServFail       byte = 2
	rcodeNxdomain       byte = 3
	rcodeNotImplemented byte = 4
	rcodeRefused        byte = 5
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
	Name  string
	Type  uint16
	Class uint16
}

type RR struct {
	Name     string
	Type     uint16
	Class    uint16
	TTL      uint32
	RDLength uint16
	RData    string
}

func parseHeader(buf [12]byte) Header {
	var header Header

	header.ID = binary.BigEndian.Uint16(buf[:2])

	flags := binary.BigEndian.Uint16(buf[2:4])
	header.QR = byte(flags & (1 << 15))
	header.Opcode = byte(flags & (0xF << 11))
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

	return header
}
