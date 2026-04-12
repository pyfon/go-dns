package main

import (
	"errors"
	"fmt"
	"net/netip"
	"strconv"

	log "github.com/sirupsen/logrus"
)

type Parser struct {
	Lexer *Lexer
	Name  string // Name of the zone file for log/err messages
}

func NewParser(l *Lexer, name string) Parser {
	return Parser{Lexer: l, Name: name}
}

func (p *Parser) Parse() (Zone, error) {
	zone := NewZone()

	// The zone name MUST be the first line of every zone.
	zoneNamed := false

	// This loop is effectively ran for every line, as handlers consume the rest of the line.
parseLoop:
	for {
		tok, err := p.Lexer.Next()
		if err != nil {
			return zone, err
		}

		switch tok.Type {
		case TokenIdent:
			record, err := p.parseRecord(tok, zone)
			if err != nil {
				return zone, err
			}
			err = zone.Insert(record)
			if err != nil {
				return zone, err
			}
		case TokenKeyword:
			if tok.Value == "zone" {
				if zoneNamed {
					errStr := fmt.Sprintf("%v Multiple zone keywords in zone file", p.Pos())
					return zone, errors.New(errStr)
				}
				zoneNamed = true
			}
			if err := p.handleKeyword(tok, &zone); err != nil {
				return zone, err
			}
		case TokenNewline:
			continue parseLoop
		case TokenEOF:
			break parseLoop
		default:
			errStr := fmt.Sprintf("%v Unexpected token: %v", p.Pos(), tok)
			return zone, errors.New(errStr)
		}
		if !zoneNamed {
			errStr := fmt.Sprintf("%v Zone name specifier must be at the top of the zone file", p.Pos())
			return zone, errors.New(errStr)
		}
	}

	return zone, nil
}

// parseRecord will parse a record line, starting with the domain name given, and return a corrisponding Record.
// zone needs to be given to properly parse a domain name,
// as it may need to be converted to a subdomain of the zone.
func (p *Parser) parseRecord(nameToken Token, zone Zone) (RData, error) {
	var record RData

	// Name (domain) field
	name := RecordName(nameToken.Value)
	if !name.Valid() {
		errStr := fmt.Sprintf("%v %v is an invalid name", p.Pos(), name)
		return record, errors.New(errStr)
	}
	record.Name = name

	// Record type field
	recTypeTok, err := p.Lexer.Next()
	if err != nil {
		return record, err
	}
	if recTypeTok.Type != TokenRecType {
		errStr := fmt.Sprintf("%v Expected a record type, got unknown value: %v", p.Pos(), recTypeTok)
		return record, errors.New(errStr)
	}
	record.Type, err = ParseRecType(recTypeTok.Value)
	if err != nil {
		return record, err
	}

	// Data/target field
	data, err := p.Lexer.Next()
	if err != nil {
		return record, err
	}
	if data.Type == TokenNewline {
		errStr := fmt.Sprintf("%v Expected data field, got newline", p.Pos())
		return record, errors.New(errStr)
	}
	// We interpret and handle the data in different ways depending on the record type.
	switch record.Type {
	case TypeA, TypeAAAA:
		if data.Type != TokenIP {
			errStr := fmt.Sprintf("%v Expected IP address, got: %v", p.Pos(), data)
			return record, errors.New(errStr)
		}
		ip, err := netip.ParseAddr(data.Value)
		if err != nil {
			return record, err
		}
		record.Addr = ip
	case TypeCNAME, TypeMX, TypeNS:
		target := Domain(data.Value)
		if !target.Valid() {
			errStr := fmt.Sprintf("%v Invalid RDATA domain: %v", p.Pos(), target)
			return record, errors.New(errStr)
		}
		if !target.FQDN() {
			target = target + "." + zone.Name.AsFQDN()
		}
		record.Target = target
	case TypeTXT:
		record.TXT = NewTXTData(data.Value)
	}

	// TTL
	ttlTok, err := p.Lexer.Next()
	if err != nil {
		return record, err
	}

	if ttlTok.Type == TokenNewline || ttlTok.Type == TokenEOF {
		return record, nil
	}
	if ttlTok.Type != TokenInt {
		errStr := fmt.Sprintf("%v Expected an integer in TTL field, got: %v", p.Pos(), ttlTok)
		return record, errors.New(errStr)
	}
	ttl, err := strconv.Atoi(ttlTok.Value)
	if err != nil {
		return record, err
	}
	if ttl <= 0 {
		errStr := fmt.Sprintf("%v TTL value cannot be <=0, got %v", p.Pos(), ttl)
		return record, errors.New(errStr)
	}
	record.TTL = uint(ttl)

	return record, nil
}

// handleKeyword handles the given keyword, consuming from the lexer as required.
// It will modify zone as required, unless an error occurs, in which case an error will be returned.
func (p *Parser) handleKeyword(keyword Token, zone *Zone) error {
	switch keyword.Value {
	case "ttl":
		return p.handleKWTTL(zone)
	case "zone":
		return p.handleKWZone(zone)
	default:
		errStr := fmt.Sprintf("%v Unexpected keyword token value: %v. This is probably a bug in the lexer.", p.Pos(), keyword)
		return errors.New(errStr)
	}
}

// handleKWZone handles the zone keyword.
// It will modify zone as required, unless an error occurs, in which case an error will be returned.
func (p *Parser) handleKWZone(zone *Zone) (err error) {
	tok, err := p.Lexer.Next()
	if err != nil {
		return err
	}
	if len(zone.Name) > 0 {
		errStr := fmt.Sprintf("%v zone domain already specifed for this zone", p.Pos())
		return errors.New(errStr)
	}
	if tok.Type != TokenIdent {
		errStr := fmt.Sprintf("%v Expected a domain after zone keyword, got: [%v]", p.Pos(), tok)
		return errors.New(errStr)
	}
	domain := Domain(tok.Value)
	if !domain.Valid() {
		errStr := fmt.Sprintf("%v Invalid domain specified for zone: %v", p.Pos(), tok.Value)
		return errors.New(errStr)
	}
	if !domain.FQDN() {
		log.Warningf("%v Zone domain is not an FQDN. Will assume it is.", p.Pos())
	}

	nl, err := p.Lexer.Next()
	if err != nil {
		return err
	}
	if nl.Type != TokenNewline && nl.Type != TokenEOF {
		errStr := fmt.Sprintf("%v Unexpected value after zone specification: %v", p.Pos(), nl.Value)
		return errors.New(errStr)
	}

	zone.Name = domain
	return nil
}

// handleKWTTL handles the ttl keyword
// It will modify zone as required, unless an error occurs, in which case an error will be returned.
func (p *Parser) handleKWTTL(zone *Zone) (err error) {
	tok, err := p.Lexer.Next()
	if err != nil {
		return err
	}
	if tok.Type != TokenInt {
		errStr := fmt.Sprintf("%v: Expected an integer after ttl keyword, got: [%v]", p.Pos(), tok)
		return errors.New(errStr)
	}
	ttl, err := strconv.Atoi(tok.Value)
	if err != nil {
		return err
	}
	if ttl <= 0 {
		errStr := fmt.Sprintf("%v TTL value cannot be <=0, got %v", p.Pos(), ttl)
		return errors.New(errStr)
	}
	zone.TTL = uint(ttl)

	nl, err := p.Lexer.Next()
	if err != nil {
		return err
	}
	if nl.Type != TokenNewline && nl.Type != TokenEOF {
		errStr := fmt.Sprintf("%v Unexpected value after ttl specification: %v", p.Pos(), nl.Value)
		return errors.New(errStr)
	}

	return nil
}

// Pos returns a short string displaying the current parser name & line number
func (p *Parser) Pos() string {
	return fmt.Sprintf("%v:%v", p.Name, p.Lexer.Line)
}
