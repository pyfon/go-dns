package main

import (
	"fmt"
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

		fmt.Printf("%v", tok)
	}

	fmt.Println("")

	return Zone{}, nil
}
