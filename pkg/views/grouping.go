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

// BuildDocTree converts a flat list of DocumentMeta into a directory tree.
// Root-level documents (no "/" in path) appear first, sorted alphabetically by path.
// Subdirectory groups appear after, sorted alphabetically by folder name.
// Supports arbitrary nesting depth via recursion.
func BuildDocTree(docs []core.DocumentMeta) []DocNode {
	if len(docs) == 0 {
		return nil
	}

	var rootDocs []core.DocumentMeta

	folderGroups := make(map[string][]core.DocumentMeta)

	for _, doc := range docs {
		slashIdx := strings.IndexByte(doc.Path, '/')
		if slashIdx == -1 {
			rootDocs = append(rootDocs, doc)
		} else {
			folder := doc.Path[:slashIdx]
			child := doc
			child.Path = doc.Path[slashIdx+1:]
			folderGroups[folder] = append(folderGroups[folder], child)
		}
	}

	sort.Slice(rootDocs, func(i, j int) bool {
		return rootDocs[i].Path < rootDocs[j].Path
	})

	nodes := make([]DocNode, 0, len(rootDocs)+len(folderGroups))

	for i := range rootDocs {
		nodes = append(nodes, DocNode{
			Doc:  &rootDocs[i],
			Name: rootDocs[i].Path,
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
			Children: BuildDocTree(folderGroups[folder]),
		})
	}

	return nodes
}
