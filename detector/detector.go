package detector

import (
	"errors"
	"io"

	"github.com/slavablind91/triego"
)

var tree triego.Trie

func init() {
	// Create a tree
	tree = triego.NewFat()

	// fill tree
	for _, s := range sigs {
		tree.Insert(s.Signature, s)
	}
}

// Detect detect input content type
func Detect(r io.Reader) (*Signature, error) {
	prev := tree
	var t triego.Trie
	var ok bool
	var b [64]byte

	for {
		n, err := r.Read(b[:])
		if err != nil {
			if err == io.EOF {
				if n > 0 {
					for _, k := range b[:n] {
						t, ok = prev.Sub(k)
						prev = nil
						if ok {
							prev = t
							continue
						}
						break
					}
					return getValue(prev)
				}
				return getValue(prev)
			}
			return nil, err
		}
		if n == 0 {
			continue
		}

		prev := tree
		for _, k := range b[:n] {
			t, ok = prev.Sub(k)
			prev = nil
			if ok {
				prev = t
				continue
			}
			return getValue(prev)
		}
	}
}

func getValue(t triego.Trie) (*Signature, error) {
	if t != nil {
		if sig := t.Value(); sig != nil {
			return sig.(*Signature), nil
		}
		return nil, errors.New("no value")
	}
	return nil, errors.New("tree is empty")
}
