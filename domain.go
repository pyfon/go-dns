package main

import (
	"regexp"
	"strings"
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

// FIXME - this needed?
// Hierarchy walks down the domain hierarchy, returning all possible parent domains for the Domain,
// ordered from the domain itself down to the top-level domain.
// For example, given the domain a.b.example.com, this function would return:
// ["a.b.example.com", "b.example.com", "example.com", "com"]
// func (d Domain) Hierarchy() []Domain {
// 	var hier []Domain

// }

// Parent returns the domains parent. For example the parent of a.example.com is "example.com".
// When the domain is already a TLD/root, the original domain will be returned and tld will be true.
func (d Domain) Parent() (domain Domain, tld bool) {
	segments := strings.Split(string(d), ".")
	if len(segments) <= 1 {
		return d, true
	}
	return Domain(strings.Join(segments[1:], ".")), false
}

func (d Domain) Root() bool {
	return d == "@"
}

func (d Domain) String() string {
	return string(d)
}
