package main

const (
	// Header QR flags
	qrQuery byte = 0
	qrReply byte = 1
	// Header opcode flags
	opcodeQuery byte = 0
	opcodeIquery byte = 1
	opcodeStatus byte = 2
	// Header response code flags
	rcodeNoError byte = 0
	rcodeFormErr byte = 1
	rcodeServFail byte = 2
	rcodeNxdomain byte = 3
	rcodeNotImplemented byte = 4
	rcodeRefused byte = 5
)

type Header struct {
	ID uint16
	QR byte // Query or reply?
	Opcode byte
	AA bool // Authoritative Answer
	TC bool // Truncated message?
	RD bool // Recursion Desired
	RA bool // Recursion Available
	// --- 1 reserved zero bit goes here ---
	AD bool // Authentic Data
	CD bool // Checking disabled
	Rcode byte
	QDCount uint16 // Number of questions
	ANCount uint16 // Number of answers
	NSCount uint16 // Number of Authority RRs
	ARCount uint16 // Number of Additional RRs
}

type Question struct {
	Name string
	Type uint16
	Class uint16
}

type RR struct {
	Name string
	Type uint16
	Class uint16
	TTL uint32
	RDLength uint16
	RData string
}
