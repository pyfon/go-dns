package main

import "strings"

// Trie is a trie data structure for domain names, to retrieve a zone or DNS records from a domain name.
// Children of the tree root will be the domain TLDs, com, biz, etc...
type Trie[T any] struct {
	root trieNode[T]
}

type trieNode[T any] struct {
	value    T
	children map[string]*trieNode[T]
}

func NewTrie[T any]() Trie[T] {
	return Trie[T]{
		root: newTrieNode[T](),
	}
}

func newTrieNode[T any]() trieNode[T] {
	return trieNode[T]{
		children: make(map[string]*trieNode[T]),
	}
}

// labelsFor will split a domain into it constituent labels. E.g. ["example", "com"]
func labelsFor(domain string) []string {
	return strings.Split(string(domain), ".")
}

// findNode will return a pointer to the trieNode for the given key.
// If it does not exist, the deepest match will be retuned, and the returned bool will be false.
// If an exact match is returned, the bool will be true.
func (t *Trie[T]) findNode(key string) (*trieNode[T], bool) {
	labels := labelsFor(key)
	node := &t.root
	for i := len(labels) - 1; i >= 0; i-- {
		child := node.children[labels[i]]
		if child == nil {
			return node, false
		}
		node = child
	}
	return node, true
}

// findCreateNode will return a pointer for the node keyed by key.
// Nodes will be created as necessary.
func (t *Trie[T]) findCreateNode(key string) *trieNode[T] {
	labels := labelsFor(key)
	node := &t.root
	for i := len(labels) - 1; i >= 0; i-- {
		child := node.children[labels[i]]
		if child == nil {
			newNode := newTrieNode[T]()
			child = &newNode
			node.children[labels[i]] = child
		}
		node = child
	}
	return node
}

// Insert will add a value to the node at the given domain. Any existing value will be replaced by the one given.
func (t *Trie[T]) Insert(key string, value T) {
	t.findCreateNode(key).value = value
}

// Search will return a pointer to the value for the given key, and a boolean indicating whether the node exists.
// If the node does not exist, the bool will be false and the deepest match will be returned, starting from the root.
func (t *Trie[T]) Search(key string) (*T, bool) {
	node, exists := t.findNode(key)
	return &node.value, exists
}


// Upsert will find or create the exact node for the given key, and pass a pointer to its value to function fn.
// Upsert will return any error returned by fn.
func (t *Trie[T]) Upsert(key string, fn func(val *T) error) error {
	node := t.findCreateNode(key)
	return fn(&node.value)
}
