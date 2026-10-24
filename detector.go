package sigo

import (
	"errors"
	"fmt"
	"io"

	"github.com/zeslava/triego"
)

var tree triego.Trie[*Signature]

func init() {
	// Create a tree
	tree = triego.NewFat[*Signature]()

	// fill tree
	for _, s := range sigs {
		tree.Insert(s.Signature, s)
	}
}

// Detect detect input content type
func Detect(r io.Reader) (*Signature, error) {
	prev := tree
	var lastValue *Signature
	var more bool
	var b [200]byte

	for {
		n, err := r.Read(b[:])
		if err != nil {
			if err == io.EOF {
				if n == 0 {
					break
				}
			} else {
				return nil, err
			}
		}
		if n == 0 {
			continue
		}

		prev, lastValue, more = iterate(b[:n], prev, lastValue)
		if more {
			continue
		}
		break
	}

	if lastValue != nil {
		return lastValue, nil
	}

	return nil, errors.New("no value")
}

func iterate(b []byte, prev triego.Trie[*Signature], lastValue *Signature) (triego.Trie[*Signature], *Signature, bool) {
	//for _, byt := range b {
	//	fmt.Printf("%#v %s\n", byt, string(byt))
	//}
	//fmt.Println()
	//for _, byt := range b {
	//	fmt.Printf("%s\t", string(byt))
	//}
	fmt.Println()

	fmt.Printf("b: %#v\n", b)
	fmt.Printf("b: %s\n", b)
	var t triego.Trie[*Signature]
	var ok bool
	for _, k := range b {
		t, ok = prev.Sub(k)
		if ok {
			prev = t
			val, _ := getValue(t)
			if val != nil {
				lastValue = val
			}
			continue
		}
		// here is no child - finish with no more
		return prev, lastValue, false
	}

	// still have child - try to get more
	return prev, lastValue, true
}

func getValue(t triego.Trie[*Signature]) (*Signature, error) {
	if t != nil {
		if sig := t.Value(); sig != nil {
			return sig, nil
		}
		return nil, errors.New("no value")
	}
	return nil, errors.New("tree is empty")
}
