package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"strconv"
	"strings"
	"unicode"
	log "github.com/sirupsen/logrus"
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

var RecordTypes = [...]RecType{
	TypeA,
	TypeAAAA,
	TypeCNAME,
	TypeTXT,
	TypeMX,
	TypeNS,
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

	var token Token
	if eof {
		token = Token{Type: TokenEOF}
	} else {
		token = l.parseValue(s)
	}

	log.Trace(token)
	return token, nil
}

// getToken reads runes from the input reader and builds a token value for
// analysis. It throws away comments and whitespace, returning a word.
// If EOF is true, input hit EOF on the first read, and no value is returned.
func (l *Lexer) getToken() (value string, EOF bool, err error) {
	var buildVal strings.Builder
	inComment := false // Whether we're currently reading a comment line.
	inQuote := false   // Whether we're currently in a "quoted string".
	escaped := false   // Whether the last rune was \

	// Read runes from input until we have built a full token for be analysed.
	for {
		r, _, err := l.input.ReadRune()
		if err != nil {
			if err == io.EOF {
				if inQuote {
					return "", true, errors.New("unterminated quoted string: hit EOF")
				}
				return "", true, nil
			} else {
				return "", false, err
			}
		}

		if r == '\n' {
			if inQuote {
				return "", false, errors.New("line ends inside a quoted string")
			}
			// Newlines are a separate token
			if buildVal.Len() == 0 {
				buildVal.WriteRune(r)
				l.Line++
				break
			}
		}

		if inComment {
			continue
		}

		if unicode.IsSpace(r) && !inQuote {
			if buildVal.Len() > 0 {
				l.input.UnreadRune()
				break
			}

			escaped = false
			continue
		}

		if r == ';' && !inQuote && !escaped {
			inComment = true
			continue
		}

		if r == '\\' && !escaped {
			escaped = true
			continue
		}

		if r == '"' && !escaped {
			inQuote = !inQuote
			continue
		}

		buildVal.WriteRune(r)
		escaped = false
	}

	return buildVal.String(), false, nil
}

// parseValue parses a string and returns a matching Token.
func (l *Lexer) parseValue(s string) Token {
	// TokenNewline
	if s == "\n" {
		return Token{Type: TokenNewline, Value: s}
	}

	// TokenInt
	if _, err := strconv.Atoi(s); err == nil {
		return Token{Type: TokenInt, Value: s}
	}

	// TokenKeyword
	if stringIsAny(s, Keywords[:]) {
		return Token{Type: TokenKeyword, Value: s}
	}

	// TokenRecType
	if stringIsAny(s, RecordTypes[:]) {
		return Token{Type: TokenRecType, Value: s}
	}

	// TokenIP
	if _, err := netip.ParseAddr(s); err == nil {
		return Token{Type: TokenIP, Value: s}
	}

	// TokenIdent
	return Token{Type: TokenIdent, Value: s}
}

// stringIsAny reports whether s equals any string in strs
func stringIsAny[T ~string](s string, strs []T) bool {
	for _, str := range strs {
		if s == string(str) {
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
