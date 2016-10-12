// Package drt implements a read-only, on-disk radix tree.
package drt

import (
	"bytes"
	"os"
	"strings"

	"github.com/SermoDigital/drt/internal/radix"
	"golang.org/x/sys/unix"

	flatbuffers "github.com/google/flatbuffers/go"
)

// Trie is a read-only, on-disk radix tree.
type Trie struct {
	root *radix.Node
	data []byte
}

// Open opens the disk-based radix trie at the given path.
func Open(path string) (*Trie, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	b, err := unix.Mmap(int(file.Fd()), 0, int(stat.Size()), unix.PROT_READ, unix.MAP_SHARED)
	if err != nil {
		return nil, err
	}

	// TODO: MADV_DONTNEED?
	err = unix.Madvise(b, unix.MADV_RANDOM)
	if err != nil {
		return nil, err
	}
	t := radix.GetRootAsTrie(b, 0)
	return &Trie{data: b, root: t.Nodes(new(radix.Node))}, nil
}

// Close closes the raidx trie.
func (t *Trie) Close() error {
	return unix.Munmap(t.data)
}

// Has returns true if the Trie contains the given key.
func (t *Trie) Has(key []byte) bool {
	node := t.root
	for len(key) != 0 {
		node, key = t.findNode(node, key)
		if node == nil {
			return false
		}
	}
	return true
}

func (t *Trie) findNode(n *radix.Node, key []byte) (*radix.Node, []byte) {
	var m radix.Node
	for i := 0; i < n.ChildrenLength(); i++ {
		if !n.Children(&m, i) {
			break
		}
		pref := m.PrefixBytes()
		if bytes.HasPrefix(key, pref) {
			return &m, key[len(pref):]
		}
	}
	return nil, key
}

// Create creates a radix trie at the given path. If one already exists it will
// be overwritten. The Writer must be closed for the radix tree to be written.
func Create(path string) (*Writer, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return &Writer{file: file, t: &trie{Root: new(node)}}, nil
}

// Close flushes the Trie to disk.
func (w *Writer) Close() error {
	b := flatbuffers.NewBuilder(0)
	_, err := w.file.Write(w.t.marshal(b))
	if err != nil {
		return err
	}
	return w.file.Close()
}

// Writer will create a read-only, on-disk radix trie.
type Writer struct {
	file *os.File
	t    *trie
}

type trie struct{ Root *node }

func (t *trie) marshal(b *flatbuffers.Builder) []byte {
	nodes := t.Root.marshal(b)
	radix.TrieStart(b)
	radix.TrieAddNodes(b, nodes)
	b.Finish(radix.TrieEnd(b))
	return b.FinishedBytes()
}

type node struct {
	Prefix   string
	Children []*node
}

func (n *node) marshal(b *flatbuffers.Builder) flatbuffers.UOffsetT {
	offs := make([]flatbuffers.UOffsetT, len(n.Children))
	for i, nv := range n.Children {
		offs[i] = nv.marshal(b)
	}
	radix.NodeStartChildrenVector(b, len(n.Children))
	for i := len(offs) - 1; i >= 0; i-- {
		b.PrependUOffsetT(offs[i])
	}
	children := b.EndVector(len(n.Children))
	pf := b.CreateString(n.Prefix)

	radix.NodeStart(b)
	radix.NodeAddPrefix(b, pf)
	radix.NodeAddChildren(b, children)
	return radix.NodeEnd(b)
}

// Insert inserts the given key into the radix trie.
func (w *Writer) Insert(key string) {
	n, match, key := w.t.Root.find(key)

	// Matched the entire key, so already inserted.
	if len(key) == 0 {
		return
	}

	// No match
	if match == 0 {
		n.Children = append(n.Children, &node{Prefix: key})
		return
	}

	// Partial match, so split the key
	common := n.Prefix[:match]
	n.Children = append(n.Children,
		&node{Prefix: strings.TrimPrefix(n.Prefix, common)},
		&node{Prefix: strings.TrimPrefix(key, common)},
	)
	n.Prefix = common
}

func (n *node) find(key string) (*node, int, string) {
	for _, nv := range n.Children {
		pl := prefixLen(key, nv.Prefix)
		// Full match
		if pl == len(nv.Prefix) {
			return nv.find(key[pl:])
		}
		// Partial match
		if pl > 0 && pl < len(nv.Prefix) {
			return nv, pl, key
		}
	}
	return n, 0, key
}

func prefixLen(s, prefix string) (n int) {
	min := len(prefix)
	if len(s) < len(prefix) {
		min = len(s)
	}
	for n < min && s[n] == prefix[n] {
		n++
	}
	return n
}
