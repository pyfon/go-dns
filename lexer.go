package main

import (
	"bufio"
	"fmt"
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
	"PTR",
}

type Token struct {
	Type  TokenType
	Value string
}

type Lexer struct {
	input *bufio.Reader
	Line  int // Current line number
}

func NewLexer(input *bufio.Reader) Lexer {
	return Lexer{input: input, Line: 1}
}

// Next scans input for the next token and returns it.
// This method will always return a TokenEOF upon EOF.
func (l *Lexer) Next() (Token, error) {
	s, eof, err := l.getToken()

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
func (l *Lexer) getToken() (value string, EOF bool, err error) {
	var buildVal strings.Builder
	inComment := false // Whether we're currently reading a comment line.

	// Read runes from input until we have built a full token for be analysed.
	for {
		r, _, err := l.input.ReadRune()
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
				l.Line++
				continue
			}

			if buildVal.Len() > 0 {
				l.input.UnreadRune()
				break
			}

			// Newline is a separate token.
			if r == '\n' {
				buildVal.WriteRune(r)
				l.Line++
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

func (t Token) String() string {
	var typeStr string
	valueStr := t.Value
	switch t.Type {
	case TokenIdent:
		typeStr = "Identifier"
	case TokenKeyword:
		typeStr = "Keyword"
	case TokenIP:
		typeStr = "IP Address"
	case TokenInt:
		typeStr = "Integer"
	case TokenRecType:
		typeStr = "Record Type"
	case TokenNewline:
		typeStr = "Newline"
		valueStr = `\n`
	case TokenEOF:
		typeStr = "EOF"
	}

	return fmt.Sprintf("%s: %s", typeStr, valueStr)
}
