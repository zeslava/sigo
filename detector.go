package sigo

import (
	"errors"
	"io"
	"sort"
	"strings"

	"github.com/zeslava/triego"
)

// ErrUnknown is returned when no signature matches the input
var ErrUnknown = errors.New("unknown signature")

// maxEmptyReads is how many consecutive 0, nil reads Detect tolerates, as in bufio
const maxEmptyReads = 100

// readSize is the size of one Read, the reads end at offset 526 + 5 at most, so head stays within two of them
const readSize = 512

// rank orders matches of different kinds, a higher rank wins regardless of length
const (
	rankFallback = iota // independent signatures at fallbackOffset and beyond, they win only if nothing else matched
	rankMain            // offset 0 and independent signatures before fallbackOffset, the longest wins
	rankChild           // signatures with After, they refine the parent
)

// fallbackOffset is where independent signatures stop competing with offset 0 by length,
// so a match at the start is not held until the stream reaches TAR at 257
const fallbackOffset = 24

// maxCursors bounds the cursors alive in one detect, roots and the children of one parent,
// so they fit an array on the stack, see TestMaxCursors
const maxCursors = 16

// cursor fields are int, not int32/int16/int8: narrower ones make the loop over bytes ~35% slower
type cursor struct {
	c      triego.Cursor[*Signature]
	offset int
	maxLen int // the longest signature in the tree
	rank   int
	dead   bool
}

var (
	// roots holds a cursor at the root of the tree per distinct offset of independent signatures, sorted by offset
	roots []cursor
	// children holds cursors of the signatures refining the parent, activated when the parent matches
	children    = map[*Signature][]cursor{}
	maxChildren int
)

type group struct {
	after  string
	offset int
}

func init() {
	groups := map[group]map[string]*Signature{}
	for _, s := range sigs {
		g := group{string(s.After), s.Offset}
		byKey := groups[g]
		if byKey == nil {
			byKey = map[string]*Signature{}
			groups[g] = byKey
		}
		key := string(s.Signature)
		if prev, ok := byKey[key]; ok {
			byKey[key] = merge(prev, s)
		} else {
			byKey[key] = s
		}
	}

	parents := groups[group{}]
	for g, byKey := range groups {
		c := newCursor(g.offset, byKey)
		switch {
		case g.after != "":
			c.rank = rankChild
			// the parent exists, see TestParents
			p := parents[g.after]
			children[p] = append(children[p], c)
			if len(children[p]) > maxChildren {
				maxChildren = len(children[p])
			}
		case g.offset >= fallbackOffset:
			c.rank = rankFallback
			roots = append(roots, c)
		default:
			c.rank = rankMain
			roots = append(roots, c)
		}
	}
	sortCursors(roots)
	for _, cs := range children {
		sortCursors(cs)
	}
}

func newCursor(offset int, byKey map[string]*Signature) cursor {
	// tree is a concrete type, not triego.Trie, so its Cursor steps are inlined
	tree := triego.New[*Signature]()
	maxLen := 0
	for _, s := range byKey {
		tree.Insert(s.Signature, s)
		if len(s.Signature) > maxLen {
			maxLen = len(s.Signature)
		}
	}
	return cursor{c: tree.Cursor(), offset: offset, maxLen: maxLen}
}

func sortCursors(cs []cursor) {
	sort.Slice(cs, func(i, j int) bool { return cs[i].offset < cs[j].offset })
}

// merge joins signatures with the same offset and bytes, so none of them is lost
func merge(a, b *Signature) *Signature {
	m := *a
	m.Description = a.Description + "|" + b.Description
	exts := strings.Split(a.Extension, "|")
	for _, e := range strings.Split(b.Extension, "|") {
		if !contains(exts, e) {
			exts = append(exts, e)
		}
	}
	m.Extension = strings.Join(exts, "|")
	return &m
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

// Detect detect input content type.
// A signature refining a matched parent wins (XLS subheader at 512 over the OLE header at 0),
// then the longest match at the start of the stream, on a tie the one at the larger offset.
// Signatures from fallbackOffset on are used only if nothing else matched
func Detect(r io.Reader) (*Signature, error) {
	s, _, _, err := detect(r, nil)
	return s, err
}

// DetectHead is Detect for streams passed on after detection: it appends the bytes taken from r
// to head and returns it, so they can be written before the rest of r, also on error.
// Pass a reused head[:0] to avoid the allocation
func DetectHead(r io.Reader, head []byte) (*Signature, []byte, error) {
	s, head, _, err := detect(r, head)
	return s, head, err
}

// detect also returns how many bytes were scanned before the result was known
func detect(r io.Reader, head []byte) (*Signature, []byte, int, error) {
	var arr [maxCursors]cursor
	cursors := arr[:len(roots)]
	copy(cursors, roots)
	alive := len(cursors)

	var best *Signature
	bestRank, bestLen, bestOffset := -1, 0, 0
	pos, empty := 0, 0

	for alive > 0 {
		if cap(head)-len(head) < readSize {
			head = append(head, make([]byte, readSize)...)[:len(head)]
		}
		b := head[len(head) : len(head)+readSize]
		n, err := r.Read(b)
		head = head[:len(head)+n]
		bn := b[:n]
		for j := 0; j < len(bn); j++ {
			k := bn[j]
			stepped := false
			for i := range cursors {
				c := &cursors[i]
				if pos < c.offset {
					break
				}
				if c.dead {
					continue
				}
				stepped = true
				if !c.c.Step(k) {
					c.dead = true
					alive--
					continue
				}
				if v := c.c.Value(); v != nil {
					l := pos - c.offset + 1
					if beats(c.rank, l, c.offset, bestRank, bestLen, bestOffset) {
						best, bestRank, bestLen, bestOffset = v, c.rank, l, c.offset
						alive -= prune(cursors, bestRank, bestLen, bestOffset)
					}
					// children start beyond pos, so they are inserted after cursor i
					if cs := children[v]; cs != nil {
						cursors = activate(cursors, cs)
						alive += len(cs)
					}
				}
			}
			pos++
			if alive == 0 {
				break
			}
			if !stepped {
				// no alive cursor is reached, so the bytes up to the nearest offset are not looked at,
				// e.g. 8..512 of an OLE file waiting for its subheaders
				skip := nextOffset(cursors, pos) - pos
				if skip > len(bn)-1-j {
					skip = len(bn) - 1 - j
				}
				j += skip
				pos += skip
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, head, pos, err
		}
		if n > 0 {
			empty = 0
		} else if empty++; empty >= maxEmptyReads {
			return nil, head, pos, io.ErrNoProgress
		}
	}

	if best == nil {
		return nil, head, pos, ErrUnknown
	}
	return best, head, pos, nil
}

// nextOffset returns the offset of the nearest alive cursor at pos or beyond, there is one while any is alive
func nextOffset(cursors []cursor, pos int) int {
	for i := range cursors {
		if c := &cursors[i]; !c.dead && c.offset >= pos {
			return c.offset
		}
	}
	return pos
}

// beats reports whether a match wins over the best one
func beats(rank, l, offset, bestRank, bestLen, bestOffset int) bool {
	if rank != bestRank {
		return rank > bestRank
	}
	return l > bestLen || l == bestLen && offset > bestOffset
}

// activate inserts cs into cursors keeping them sorted by offset
func activate(cursors, cs []cursor) []cursor {
	for _, c := range cs {
		j := len(cursors)
		for j > 0 && cursors[j-1].offset > c.offset {
			j--
		}
		cursors = append(cursors, cursor{})
		copy(cursors[j+1:], cursors[j:])
		cursors[j] = c
	}
	return cursors
}

// prune kills cursors that can no longer beat the best match even with their longest signature,
// so Detect does not read up to offsets that cannot change the result
func prune(cursors []cursor, bestRank, bestLen, bestOffset int) int {
	killed := 0
	for i := range cursors {
		c := &cursors[i]
		if c.dead || beats(c.rank, c.maxLen, c.offset, bestRank, bestLen, bestOffset) {
			continue
		}
		c.dead = true
		killed++
	}
	return killed
}
