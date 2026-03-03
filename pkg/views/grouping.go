// Package views provides HTML template rendering for the documentation portal.
package views

import (
	"sort"
	"strings"

	"github.com/ksysoev/omnidex/pkg/core"
)

// DocNode represents either a document or a folder in the directory tree.
// When Doc is non-nil, this is a leaf document node and Children is empty.
// When Doc is nil, this is a folder node and Children holds the subtree.
type DocNode struct {
	Doc      *core.DocumentMeta
	Name     string
	Children []DocNode
}

// docEntry pairs an original DocumentMeta (full path preserved) with the
// remaining path suffix used purely for recursive grouping.
type docEntry struct {
	doc           *core.DocumentMeta
	remainingPath string
}

// BuildDocTree converts a flat list of DocumentMeta into a directory tree.
// Root-level documents (no "/" in path) appear first, sorted alphabetically by path.
// Subdirectory groups appear after, sorted alphabetically by folder name.
// Supports arbitrary nesting depth via recursion.
// The original DocumentMeta values (including full paths) are never mutated.
func BuildDocTree(docs []core.DocumentMeta) []DocNode {
	if len(docs) == 0 {
		return nil
	}

	entries := make([]docEntry, len(docs))
	for i := range docs {
		entries[i] = docEntry{doc: &docs[i], remainingPath: docs[i].Path}
	}

	return buildTree(entries)
}

// buildTree is the internal recursive helper that operates on docEntry slices
// so original DocumentMeta pointers (and their full paths) are never modified.
func buildTree(entries []docEntry) []DocNode {
	if len(entries) == 0 {
		return nil
	}

	var rootEntries []docEntry

	folderGroups := make(map[string][]docEntry)

	for _, e := range entries {
		slashIdx := strings.IndexByte(e.remainingPath, '/')
		if slashIdx == -1 {
			rootEntries = append(rootEntries, e)
		} else {
			folder := e.remainingPath[:slashIdx]
			child := e
			child.remainingPath = e.remainingPath[slashIdx+1:]
			folderGroups[folder] = append(folderGroups[folder], child)
		}
	}

	sort.Slice(rootEntries, func(i, j int) bool {
		return rootEntries[i].doc.Path < rootEntries[j].doc.Path
	})

	nodes := make([]DocNode, 0, len(rootEntries)+len(folderGroups))

	for _, e := range rootEntries {
		nodes = append(nodes, DocNode{
			Doc:  e.doc,
			Name: e.remainingPath,
		})
	}

	folders := make([]string, 0, len(folderGroups))
	for f := range folderGroups {
		folders = append(folders, f)
	}

	sort.Strings(folders)

	for _, folder := range folders {
		nodes = append(nodes, DocNode{
			Name:     folder,
			Children: buildTree(folderGroups[folder]),
		})
	}

	return nodes
}
