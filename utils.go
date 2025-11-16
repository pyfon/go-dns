package main

// FindBestZoneMatch finds the zone which is the most specific match for domain in the zone map
// and returns a pointer to it.
// For example a.b.example.com would first match the b.example.com zone if present, if not example.com, if not com.
// This function will return nil if no match is found in the zones map.
func FindBestZoneMatch(zones map[Domain]*Zone, domain Domain) *Zone {
	curDomain := domain
	for {
		zone, ok := zones[curDomain]
		if ok {
			return zone
		}
		var tld bool
		curDomain, tld = curDomain.Parent()
		if tld {
			return nil // We've hit the root with no matches.
		}
	}
}
