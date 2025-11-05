package main

import (
	"bufio"
	"io"
	"net/netip"
	"strconv"
	"strings"
	"unicode"
)

type TokenType int

const (
	TokenIdent TokenType = iota
	TokenKeyword
	TokenIP
	TokenInt
	TokenRecType // A, AAAA, etc...
	TokenNewline
	TokenEOF
)

var Keywords = [...]string{
	"zone",
	"ttl",
}

var RecordTypes = [...]string{
	"A",
	"AAAA",
	"CNAME",
	"TXT",
	"MX",
	"NS",
}

type Token struct {
	Type  TokenType
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
	s, eof, err := getToken(l.input)

	if err != nil {
		return Token{}, err
	}

	// TokenEOF
	if eof {
		return Token{Type: TokenEOF}, nil
	}

	// TokenNewline
	if s == "\n" {
		return Token{Type: TokenNewline, Value: s}, nil
	}

	// TokenInt
	if _, err := strconv.Atoi(s); err == nil {
		return Token{Type: TokenInt, Value: s}, nil
	}

	// TokenKeyword
	if stringIsAny(s, Keywords[:]) {
		return Token{Type: TokenKeyword, Value: s}, nil
	}

	// TokenRecType
	if stringIsAny(s, RecordTypes[:]) {
		return Token{Type: TokenRecType, Value: s}, nil
	}

	// TokenIP
	if _, err := netip.ParseAddr(s); err == nil {
		return Token{Type: TokenIP, Value: s}, nil
	}

	// TokenIdent
	return Token{Type: TokenIdent, Value: s}, nil
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

			// Newline is a separate token.
			if r == '\n' {
				buildVal.WriteRune(r)
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

// stringIsAny reports whether s equals any string in strs
func stringIsAny(s string, strs []string) bool {
	for _, str := range strs {
		if s == str {
			return true
		}
	}
	return false
}
