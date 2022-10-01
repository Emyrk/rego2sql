package go_rego

import (
	"fmt"
	"regexp"
	"strings"
)

type Tree struct {
	Root *Node
}

type Node struct {
	Children map[string]*Node

	// Values to affect conversion
	NodeSQLType SQLType
	ColumnName  ColumnNameFunc
}

type ColumnNameFunc func(path []string) string

func StaticName(name string) ColumnNameFunc {
	return func(path []string) string {
		return name
	}
}

func RegexColumnNameReplace(regexStr, replaceStr string) ColumnNameFunc {
	re := regexp.MustCompile(regexStr)
	return func(path []string) string {
		str := strings.Join(path, ".")
		matches := re.FindStringSubmatch(str)
		if len(matches) > 0 {
			// This config matches this variable.
			replace := make([]string, 0, len(matches)*2)
			for i, m := range matches {
				replace = append(replace, fmt.Sprintf("$%d", i))
				replace = append(replace, m)
			}
			replacer := strings.NewReplacer(replace...)
			return replacer.Replace(replaceStr)
		}
		return "UNKNOWN"
	}
}

func NewTree() *Tree {
	return &Tree{
		Root: &Node{
			Children: make(map[string]*Node),
		},
	}
}

func (t *Tree) PathNode(path []string) (*Node, []string) {
	return t.Root.PathNode(path)
}

func (t *Tree) AddElement(path []string, sqlType SQLType, nameFunc ColumnNameFunc) *Tree {
	t.Root.AddChild(path, sqlType, nameFunc)
	return t
}

func (n *Node) AddChild(path []string, sqlType SQLType, nameFunc ColumnNameFunc) {
	if len(path) == 0 {
		n.NodeSQLType = sqlType
		n.ColumnName = nameFunc
		return
	}

	nextKey := path[0]
	next, ok := n.Children[nextKey]
	if !ok {
		n.Children[nextKey] = &Node{
			Children: make(map[string]*Node),
		}
		next = n.Children[nextKey]
	}
	next.AddChild(path[1:], sqlType, nameFunc)
}

func (t *Node) PathNode(path []string) (*Node, []string) {
	if len(path) == 0 {
		return t, path
	}
	next := t.Children[path[0]]
	if next == nil {
		return t, path
	}
	return next.PathNode(path[1:])
}
