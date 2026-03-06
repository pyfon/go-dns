package main

import "testing"

// WIP: quick test.
func TestTrie(t *testing.T) {
	trie := NewTrie[string]()
	key := Domain("my.test.domain.example.com")
	val := "This is my test string 123"
	trie.Insert(key, val)
	retrieved, found := trie.Search(key)
	if !found {
		t.Errorf("Trie didn't find expected value")
	}
	if retrieved != val {
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

func TestZoneTrie(t *testing.T) {
	zones := testZones()
	zoneTrie := NewZoneTrie(zones)

	for domain, zone := range zones {
		result, exists := zoneTrie.Search(domain)
		if !exists {
			t.Errorf("Domain %s does not exist in the trie", domain)
		}
		if result.Name != zone.Name {
			t.Errorf("Wrong zone returned from zone. Expected zone %s, got zone %s", zone.Name, result.Name)
		}
	}
}
