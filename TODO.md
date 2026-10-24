# TODO

## sigo

### SVG
Text format with no binary signature (`<?xml` / `<svg`), its test is skipped.

### OLE subtypes
DOC/XLS/PPT are detected only as the generic OLE header (16 merged types): the examples have
`fd ff ff ff ff ff ff ff` at 512, no subheader matches. A real subtype needs parsing the OLE directory.

## triego

- `Compact`: immutable trie with all nodes in one slice in BFS order, built from `trie` by `Compile`.
  On a 104k words dictionary Find is -25..-32% and memory is 15.2MB vs 26.6MB;
  no gain on sigo signatures, the tree is in cache anyway. Worth it only for large trees
- `Compact` node of 48 bytes instead of 64: value index instead of value, needs `comparable` or a flag
- `Delete` does not prune empty ancestors
- key vs prefix: zero value means "no key ends here", so a key with a zero value is not distinguishable
