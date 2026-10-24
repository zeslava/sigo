package sigo

import (
	"errors"
	"io"

	"github.com/zeslava/triego"
)

// tree is a concrete type, not triego.Trie, so its Cursor steps are inlined
var tree = triego.New[*Signature]()

func init() {
	// fill tree
	for _, s := range sigs {
		tree.Insert(s.Signature, s)
	}
}

// Detect detect input content type
func Detect(r io.Reader) (*Signature, error) {
	c := tree.Cursor()
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

		lastValue, more = iterate(b[:n], &c, lastValue)
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

func iterate(b []byte, c *triego.Cursor[*Signature], lastValue *Signature) (*Signature, bool) {
	for _, k := range b {
		if !c.Step(k) {
			// here is no child - finish with no more
			return lastValue, false
		}
		if val := c.Value(); val != nil {
			lastValue = val
		}
	}

	// still have child - try to get more
	return lastValue, true
}
