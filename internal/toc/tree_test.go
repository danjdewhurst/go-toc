package toc

import (
	"testing"
)

func TestNewNode(t *testing.T) {
	node := NewNode("test.md", "docs/test.md", false)

	if node.Name != "test.md" {
		t.Errorf("expected Name 'test.md', got '%s'", node.Name)
	}
	if node.Path != "docs/test.md" {
		t.Errorf("expected Path 'docs/test.md', got '%s'", node.Path)
	}
	if node.IsDir {
		t.Error("expected IsDir to be false")
	}
	if len(node.Children) != 0 {
		t.Errorf("expected empty Children, got %d", len(node.Children))
	}
}

func TestNodeAddChild(t *testing.T) {
	parent := NewNode("docs", "docs", true)
	child := NewNode("test.md", "docs/test.md", false)

	result := parent.AddChild(child)

	if len(parent.Children) != 1 {
		t.Errorf("expected 1 child, got %d", len(parent.Children))
	}
	if result != child {
		t.Error("AddChild should return the added child")
	}
}

func TestNodeSort(t *testing.T) {
	root := NewNode("root", ".", true)

	// Add in unsorted order
	root.AddChild(NewNode("zebra.md", "zebra.md", false))
	root.AddChild(NewNode("alpha", "alpha", true))
	root.AddChild(NewNode("apple.md", "apple.md", false))
	root.AddChild(NewNode("beta", "beta", true))

	root.Sort()

	// Directories should come first, then files, both alphabetically
	expected := []string{"alpha", "beta", "apple.md", "zebra.md"}
	for i, name := range expected {
		if root.Children[i].Name != name {
			t.Errorf("position %d: expected '%s', got '%s'", i, name, root.Children[i].Name)
		}
	}
}

func TestNodeFindOrCreatePath(t *testing.T) {
	root := NewNode("root", ".", true)

	// Create nested path
	node := root.FindOrCreatePath("docs/api/handlers.md", false)

	if node.Name != "handlers.md" {
		t.Errorf("expected Name 'handlers.md', got '%s'", node.Name)
	}
	if node.IsDir {
		t.Error("expected final node to be a file")
	}

	// Verify intermediate directories were created
	if len(root.Children) != 1 {
		t.Fatal("expected 1 child at root")
	}
	docsNode := root.Children[0]
	if docsNode.Name != "docs" {
		t.Errorf("expected 'docs', got '%s'", docsNode.Name)
	}
	if !docsNode.IsDir {
		t.Error("expected docs to be a directory")
	}
}

func TestTree(t *testing.T) {
	tree := NewTree("project")

	tree.AddFile("README.md")
	tree.AddFile("docs/guide.md")
	tree.AddDirectory("src")

	tree.Sort()

	// Verify structure
	if len(tree.Root.Children) != 3 {
		t.Errorf("expected 3 children, got %d", len(tree.Root.Children))
	}
}

func TestTreeWalk(t *testing.T) {
	tree := NewTree("project")

	tree.AddFile("README.md")
	tree.AddFile("docs/guide.md")
	tree.AddFile("docs/api.md")

	tree.Sort()

	var visited []string
	tree.Walk(func(node *Node, depth int, isLast bool) {
		visited = append(visited, node.Name)
	})

	// Expect 4 nodes: docs (dir), api.md, guide.md, README.md
	if len(visited) != 4 {
		t.Errorf("expected 4 nodes visited, got %d: %v", len(visited), visited)
	}
}
