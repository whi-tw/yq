package yqlib

import (
	"container/list"
	"fmt"
	"reflect"

	ast "github.com/goccy/go-yaml/ast"
	"github.com/jinzhu/copier"
)

type CandidateNode struct {
	Node   ast.Node       // the actual node
	Key    ast.Node       // key, if this is an entry in a map
	Index  int            // index, if this is an entry in an array
	Parent *CandidateNode // parent node

	LeadingContent string

	Path      []interface{} /// the path we took to get to this node
	Document  uint          // the document index of this node
	Filename  string
	FileIndex int
	// when performing op against all nodes given, this will treat all the nodes as one
	// (e.g. top level cross document merge). This property does not propegate to child nodes.
	EvaluateTogether bool
	IsMapKey         bool
}

func (n *CandidateNode) GetKey() string {
	keyPrefix := ""
	if n.IsMapKey {
		keyPrefix = "key-"
	}
	return fmt.Sprintf("%v%v - %v", keyPrefix, n.Document, n.Path)
}

func (n *CandidateNode) AsList() *list.List {
	elMap := list.New()
	elMap.PushBack(n)
	return elMap
}

func (n *CandidateNode) CreateMapChild(key ast.Node, value ast.Node) *CandidateNode {
	return &CandidateNode{
		Node:      value,
		Path:      n.createChildPath(key.String()),
		Key:       key,
		Parent:    n,
		Document:  n.Document,
		Filename:  n.Filename,
		FileIndex: n.FileIndex,
	}
}

func (n *CandidateNode) CreateArrayChild(index int, value ast.Node) *CandidateNode {
	return &CandidateNode{
		Node:      value,
		Path:      n.createChildPath(index),
		Index:     index,
		Parent:    n,
		Document:  n.Document,
		Filename:  n.Filename,
		FileIndex: n.FileIndex,
	}
}

func (n *CandidateNode) CreateReplacementCandidate(value ast.Node) *CandidateNode {
	return &CandidateNode{
		Node:      value,
		Path:      n.createChildPath(nil),
		Parent:    n.Parent,
		Key:       n.Key,
		Index:     n.Index,
		Document:  n.Document,
		Filename:  n.Filename,
		FileIndex: n.FileIndex,
	}
}

// func (n *CandidateNode) CreateChild(path interface{}, node ast.Node) *CandidateNode {
// 	parent := n
// 	if path == nil {
// 		parent = n.Parent
// 	}
// 	return &CandidateNode{
// 		Node:      node,
// 		Path:      n.createChildPath(path),
// 		Parent:    parent,
// 		Document:  n.Document,
// 		Filename:  n.Filename,
// 		FileIndex: n.FileIndex,
// 	}
// }

func (n *CandidateNode) createChildPath(path interface{}) []interface{} {
	if path == nil {
		newPath := make([]interface{}, len(n.Path))
		copy(newPath, n.Path)
		return newPath
	}

	//don't use append as they may actually modify the path of the orignal node!
	newPath := make([]interface{}, len(n.Path)+1)
	copy(newPath, n.Path)
	newPath[len(n.Path)] = path
	return newPath
}

func (n *CandidateNode) Copy() (*CandidateNode, error) {
	clone := &CandidateNode{}
	err := copier.Copy(clone, n)
	if err != nil {
		return nil, err
	}
	return clone, nil
}

func (n *CandidateNode) ReplaceWith(node ast.Node) error {
	n.Node = node

	if n.IsMapKey {
		return fmt.Errorf("cannot replace map keys yet")
	}

	switch n.Parent.Node.(type) {
	case *ast.MappingNode:
		parent := n.Parent.Node.(*ast.MappingNode)
		_, err := ReplaceInMap(parent, n.Key, node)
		return err
	case *ast.SequenceNode:
		parent := n.Parent.Node.(*ast.SequenceNode)
		return parent.Replace(n.Index, node)
	}
	return fmt.Errorf("can only replace maps and seqs, %v", reflect.TypeOf(n.Parent.Node))
}

// updates this candidate from the given candidate node
func (n *CandidateNode) Merge(other *CandidateNode) ast.Node {

	// n.UpdateAttributesFrom(other)
	// need to do this at the parent level with goccy
	// as it involves changing the type!
	// no update -immutable

	// n.Node.Content = other.Node.Content
	// n.Node.Value = other.Node.Value
	return n.MergeAttributes(other)
}

func (n *CandidateNode) MergeAttributes(other *CandidateNode) ast.Node {
	log.Debug("UpdateAttributesFrom: n: %v other: %v", n.GetKey(), other.GetKey())
	return &ast.StringNode{Value: "new thing!"}
	// var replacement ast.Node =
	// if n.Node.Type() != other.Node.Type() {
	// 	replacement = other.Node.
	// }
	// target = &CandidateNode{
	// 	Parent: n.Parent,
	// 	Path:   n.Path,
	// }

	// if reflect.TypeOf(n.Node) !=

	// if n.Node.Kind != other.Node.Kind {
	// 	// clear out the contents when switching to a different type
	// 	// e.g. map to array
	// 	n.Node.Content = make([]*yaml.Node, 0)
	// 	n.Node.Value = ""
	// }
	// n.Node.Kind = other.Node.Kind
	// n.Node.Tag = other.Node.Tag
	// n.Node.Alias = other.Node.Alias
	// n.Node.Anchor = other.Node.Anchor

	// // merge will pickup the style of the new thing
	// // when autocreating nodes
	// if n.Node.Style == 0 {
	// 	n.Node.Style = other.Node.Style
	// }

	// if other.Node.FootComment != "" {
	// 	n.Node.FootComment = other.Node.FootComment
	// }
	// if other.Node.HeadComment != "" {
	// 	n.Node.HeadComment = other.Node.HeadComment
	// }
	// if other.Node.LineComment != "" {
	// 	n.Node.LineComment = other.Node.LineComment
	// }
}
