package detector

import "sort"

type trie struct {
	root *node
}

func (t *trie) insert(path []byte, value interface{}) *node {
	if t.root == nil {
		t.root = &node{}
	}

	n := t.root
	for _, k := range path {
		n = n.insChild(k)
	}
	n.set(value)
	return n
}

func (t *trie) delete(path []byte) {
	var ch *node
	for i := range path[:len(path)-1] {
		if ch = t.root.getChild(path[i]); ch == nil {
			return
		}
	}
	ch.delChild(path[len(path)-1])
}

func (t *trie) get(path []byte) interface{} {
	n := t.root
	for _, k := range path {
		if child := n.getChild(k); child != nil {
			n = child
		} else {
			return n.value
		}
	}
	if n.value != nil {
		return n.value
	}

	return nil
}

func (t *trie) set(path []byte, value interface{}) {
	_ = t.insert(path, value)
}

type node struct {
	key   byte
	value interface{}
	nodes []*node
}

func (n *node) insChild(key byte) *node {
	if n.nodes == nil {
		n.nodes = make([]*node, 0, 1)
	}

	v := n.getChild(key)
	if v != nil {
		return v
	}

	v = &node{key: key}
	i := sort.Search(len(n.nodes), func(i int) bool { return (n.nodes)[i].key >= key })
	if i == len(n.nodes) {
		n.nodes = append(n.nodes, v)
	} else {
		n.nodes = append(n.nodes, nil)
		copy(n.nodes[i+1:], n.nodes[i:])
		(n.nodes)[i] = v
	}
	return v
}

func (n *node) getChild(key byte) *node {
	i := sort.Search(len(n.nodes), func(i int) bool { return (n.nodes)[i].key >= key })
	if i < len(n.nodes) && (n.nodes)[i].key == key {
		return n.nodes[i]
	}
	return nil
}

func (n *node) delChild(key byte) {
	num := len(n.nodes)
	i := sort.Search(num, func(i int) bool { return n.nodes[i].key >= key })
	if i < len(n.nodes) && (n.nodes)[i].key == key {
		copy(n.nodes[i:], n.nodes[i+1:])
		n.nodes[len(n.nodes)-1] = nil
		n.nodes = n.nodes[:len(n.nodes)-1]
	}
}

func (n *node) get() interface{} {
	return n.value
}

func (n *node) set(value interface{}) {
	n.value = value
}
