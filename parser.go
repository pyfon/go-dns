package main

import (
	"errors"
	"fmt"
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
	var zone Zone

	// This loop is effectively ran for every line, as handlers consume the rest of the line.
parseLoop:
	for {
		tok, err := p.Lexer.Next()
		if err != nil {
			return Zone{}, err
		}

		switch tok.Type {
		case TokenIdent:
			record, err := p.parseRecord(tok)
			if err != nil {
				return zone, err
			}
			zone.Records[record.Name] = record
		case TokenKeyword:
			if err := p.handleKeyword(tok, &zone); err != nil {
				return zone, err
			}
		case TokenNewline:
			continue parseLoop
		case TokenEOF:
			break parseLoop
		default:
			errStr := fmt.Sprintf("%v Unexpected token: %v", p.Pos(), tok)
			return Zone{}, errors.New(errStr)
		}
	}

	return zone, nil
}

// parseRecord will parse a record line, starting with the domain name given, and return a corrisponding Record.
func (p *Parser) parseRecord(name Token) (Record, error) {
	var record Record
	// --- TODO ---
	return record, nil
}

// handleKeyword handles the given keyword, consuming from the lexer as required.
// It will modify zone as required, unless an error occurs, in which case an error will be returned.
func (p *Parser) handleKeyword(keyword Token, zone *Zone) (err error) {
	switch keyword.Value {
	case "zone":
		return p.handleKWZone(zone)
	case "ttl":
		return p.handleKWTTL(zone)
	default:
		errStr := fmt.Sprintf("Unexpected keyword token value: %v. This is probably a bug in the lexer.", keyword)
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
	if len(zone.Zone) > 0 {
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

	zone.Zone = domain
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

	zone.TTL = ttl
	return nil
}

// Pos returns a short string displaying the current parser name & line number
func (p *Parser) Pos() string {
	return fmt.Sprintf("%v:%v", p.Name, p.Lexer.Line)
}
