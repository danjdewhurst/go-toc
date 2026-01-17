package toc

import (
	"path/filepath"
	"sort"
	"strings"
)

// Node represents a file or directory in the tree structure.
type Node struct {
	Name       string           // File or directory name
	Path       string           // Relative path from root
	IsDir      bool             // True if this is a directory
	Summary    string           // First paragraph summary (for markdown files)
	Children   []*Node          // Child nodes (for directories)
	childIndex map[string]*Node // Fast lookup of children by name
}

// NewNode creates a new tree node.
func NewNode(name, path string, isDir bool) *Node {
	return &Node{
		Name:       name,
		Path:       path,
		IsDir:      isDir,
		Children:   make([]*Node, 0),
		childIndex: make(map[string]*Node),
	}
}

// AddChild adds a child node and returns it.
func (n *Node) AddChild(child *Node) *Node {
	n.Children = append(n.Children, child)
	n.childIndex[child.Name] = child
	return child
}

// Sort sorts children recursively. Directories come first, then files.
// Within each group, items are sorted alphabetically.
func (n *Node) Sort() {
	sort.Slice(n.Children, func(i, j int) bool {
		// Directories first
		if n.Children[i].IsDir != n.Children[j].IsDir {
			return n.Children[i].IsDir
		}
		// Then alphabetically (case-insensitive)
		return strings.ToLower(n.Children[i].Name) < strings.ToLower(n.Children[j].Name)
	})

	// Sort children recursively
	for _, child := range n.Children {
		child.Sort()
	}
}

// FindOrCreatePath finds or creates the path in the tree, returning the final node.
// Uses O(1) map lookup for children instead of O(n) linear search.
func (n *Node) FindOrCreatePath(relPath string, isDir bool) *Node {
	if relPath == "" || relPath == "." {
		return n
	}

	parts := strings.Split(filepath.ToSlash(relPath), "/")
	current := n

	for i, part := range parts {
		if part == "" {
			continue
		}

		// O(1) map lookup instead of O(n) linear search
		if child, exists := current.childIndex[part]; exists {
			current = child
		} else {
			// Determine if this part is a directory
			// It's a directory if it's not the last part, or if isDir is true for the last part
			partIsDir := i < len(parts)-1 || isDir
			partPath := strings.Join(parts[:i+1], "/")
			newNode := NewNode(part, partPath, partIsDir)
			current.AddChild(newNode)
			current = newNode
		}
	}

	return current
}

// Tree represents the complete file tree structure.
type Tree struct {
	Root *Node
}

// NewTree creates a new tree with the given root name.
func NewTree(rootName string) *Tree {
	return &Tree{
		Root: NewNode(rootName, ".", true),
	}
}

// AddFile adds a file to the tree at the specified relative path.
func (t *Tree) AddFile(relPath string) *Node {
	return t.Root.FindOrCreatePath(relPath, false)
}

// AddDirectory adds a directory to the tree at the specified relative path.
func (t *Tree) AddDirectory(relPath string) *Node {
	return t.Root.FindOrCreatePath(relPath, true)
}

// Sort sorts the entire tree.
func (t *Tree) Sort() {
	t.Root.Sort()
}

// Walk traverses the tree in depth-first order, calling fn for each node.
// The depth parameter indicates the nesting level (0 for root's children).
func (t *Tree) Walk(fn func(node *Node, depth int, isLast bool)) {
	walkNode(t.Root.Children, 0, fn)
}

func walkNode(nodes []*Node, depth int, fn func(node *Node, depth int, isLast bool)) {
	for i, node := range nodes {
		isLast := i == len(nodes)-1
		fn(node, depth, isLast)
		if node.IsDir && len(node.Children) > 0 {
			walkNode(node.Children, depth+1, fn)
		}
	}
}
