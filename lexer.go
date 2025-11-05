package main

import (
	"bufio"
	"io"
	"unicode"
	"strings"
)

type TokenType int

const (
	TokenIdent TokenType = iota
	TokenKeyword
	TokenIP
	TokenTTLTime // e.g. 5m
	TokenRecType // A, AAAA, etc...
	TokenEOF
)

type Token struct {
	Type TokenType
	Value string
}

type Lexer struct {
	input *bufio.Reader
}

func NewLexer(input *bufio.Reader) Lexer {
	return Lexer{input}
}

// Next scans input for the next token and returns it.
// This method will always return a TokenEOF upon EOF.
func (l *Lexer) Next() (Token, error) {
	val, eof, err := getToken(l.input)
	if err != nil {
		return Token{}, err
	}
	if eof {
		return Token{Type: TokenEOF}, nil
	}

	// TODO: analyse the value, return TokenIP, TTLTime, RecType.

	if val == "zone" || val == "ttl" {
		return Token{Type: TokenKeyword, Value: val}, nil
	}

	return Token{Type: TokenIdent, Value: val}, nil
}

// getToken reads runes from the input reader and builds a token value for
// analysis. It throws away comments and whitespace, returning a word.
// If EOF is true, input hit EOF on the first read, and no value is returned.
func getToken(input *bufio.Reader) (value string, EOF bool, err error) {
	var buildVal strings.Builder
	inComment := false // Whether we're currently reading a comment line.

	// Read runes from input until we have built a full token for be analysed.
	for {
		r, _, err := input.ReadRune()
		if err != nil {
			if err == io.EOF {
				return "", true, nil
			} else {
				return "", false, err
			}
		}

		if unicode.IsSpace(r) {
			if inComment && r == '\n' {
				inComment = false
				continue
			}

			if buildVal.Len() > 0 {
				input.UnreadRune()
				break
			}
			continue
		}

		if inComment {
			continue
		}

		if r == ';' {
			inComment = true
			continue
		}

		buildVal.WriteRune(r)
	}

	return buildVal.String(), false, nil
}
