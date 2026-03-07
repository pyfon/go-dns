package main

import "testing"

func TestTrie(t *testing.T) {
	trie := NewTrie[string]()
	key := "my.test.domain.example.com"
	val := "This is my test string 123"
	trie.Insert(key, val)
	retrieved, found := trie.Search(key)
	if !found {
		t.Errorf("Trie didn't find expected value")
	}
	if *retrieved != val {
		t.Errorf("Trie didn't retrieve the expected value")
	}
}

func testZones() map[Domain]Zone {
	zoneMap := make(map[Domain]Zone)
	domains := []Domain{
		"example.com",
		"x.example.com",
		"w.x.example.com",
		"wow.biz",
		"com",
	}
	for _, domain := range domains {
		zone := NewZone()
		zone.Name = domain
		zoneMap[domain] = zone
	}
	return zoneMap
}

// TestZoneTrieExact ensures that we get the zone which exactly matches the search argument.
func TestZoneTrieExact(t *testing.T) {
	zones := testZones()
	zoneTrie := NewZoneTrie(zones)

	for domain, zone := range zones {
		result, exists := zoneTrie.Search(string(domain))
		if !exists {
			t.Errorf("Domain %s does not exist in the trie", domain)
		}
		if result.Name != zone.Name {
			t.Errorf("Wrong zone returned from zone. Expected zone %s, got zone %s", zone.Name, result.Name)
		}
	}
}

// TestZoneTrieClosest ensures that we get the closest ancestor of the a searched domain if there isn't an exact match.
func TestZoneTrieClosest(t *testing.T) {
	zones := testZones()
	zoneTrie := NewZoneTrie(zones)
	domain := Domain("other.x.example.com")
	zone, exists := zoneTrie.Search("other.x.example.com")
	if exists {
		t.Errorf("%s exists in the tree for some reason.", domain)
	}
	expected := Domain("x.example.com")
	if zone.Name != expected {
		t.Errorf("Didn't get closest ancestor of %s, Expected: %s. Got: %s", domain, expected, zone.Name)
	}
}

func TestZoneTrieEmptyKey(t *testing.T) {
	zones := testZones()
	zoneTrie := NewZoneTrie(zones)
	myZone := NewZone()
	myZone.Name = "ROOT"
	zoneTrie.Insert("", myZone)

	result, exists := zoneTrie.Search("")
	if !exists {
		t.Errorf("Expected zone at root (empty key) to exist")
	}
	if result.Name != myZone.Name {
		t.Errorf("Root node exists but it did not contain the inserted zone")
	}
}
