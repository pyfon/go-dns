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
