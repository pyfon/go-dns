package main

import "strings"

// Trie is a trie data structure for domain names, to retrieve a zone or DNS records from a domain name.
// Children of the tree root will be the domain TLDs, com, biz, etc...
type Trie[T any] struct {
	root trieNode[T]
}

type trieNode[T any] struct {
	value T
	children map[string]*trieNode[T]
}

func NewTrie[T any]() Trie[T] {
	return Trie[T] {
		root: newTrieNode[T](),
	}
}

func newTrieNode[T any]() trieNode[T] {
	return trieNode[T] {
		children: make(map[string]*trieNode[T]),
	}
}

// Insert will add a value to the node at the given domain. Any existing value will be replaced by the one given.
func (t *Trie[T]) Insert(key Domain, value T) {
	labels := strings.Split(string(key), ".")
	// Iterate through the DNS in reverese order so we can fill out the tree from the root.
	node := &t.root
	for i := len(labels) - 1; i >= 0; i-- {
		child :=  node.children[labels[i]]
		if child == nil {
			newNode := newTrieNode[T]()
			child = &newNode
			node.children[labels[i]] = child
		}
		node = child
	}
	node.value = value
}

// Search will retrieve the value for the given key, and a boolean indicating whether the node exists.
// If the node does not exist, the bool will be false. Otherwise, it will be true.
func (t *Trie[T]) Search(key Domain) (T, bool) {
	labels := strings.Split(string(key), ".")
	node := &t.root
	for i := len(labels) - 1; i >= 0; i-- {
		node =  node.children[labels[i]]
		if node == nil {
			var zero T
			return zero, false
		}
	}
	return node.value, true
}

// func NewZoneTrie(zones []Zone) Trie[Zone] {
// 	var trie Trie[Zone]
// 	for _, zone := range zones {
// 	}
// }
