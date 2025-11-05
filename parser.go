package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
)

type Parser struct {
	Lexer *Lexer
}

func NewParser(l *Lexer) Parser {
	return Parser{Lexer: l}
}

func (p *Parser) Parse() (Zone, error) {
	for {
		tok, err := p.Lexer.Next()
		if err != nil {
			return Zone{}, err
		}

		if tok.Type == TokenEOF {
			break
		}

		log.Tracef("Lexer gave token: %v\n", tok)
	}

	fmt.Println("")

	return Zone{}, nil
}
