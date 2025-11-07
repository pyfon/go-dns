package main

import (
	"regexp"
)

var domainRegex *regexp.Regexp = regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.?)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9]\.?$`)

type Domain string

// Valid reports whether the domain is a valid domain.
func (d Domain) Valid() bool {
	if d == "@" {
		return true
	}
	return domainRegex.MatchString(string(d))
}

// FQDN reports whether Domain is fully-qualified. It does not check for domain validity.
func (d Domain) FQDN() bool {
	return d[len(d)-1:] == "."
}

func (d Domain) Root() bool {
	return d == "@"
}

func (d Domain) String() string {
	return string(d)
}
