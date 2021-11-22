package yqlib

import (
	"container/list"

	"github.com/goccy/go-yaml/ast"
)

func getDocumentIndexOperator(d *dataTreeNavigator, context Context, expressionNode *ExpressionNode) (Context, error) {
	var results = list.New()

	for el := context.MatchingNodes.Front(); el != nil; el = el.Next() {
		candidate := el.Value.(*CandidateNode)
		node := &ast.IntegerNode{Value: candidate.Document}
		scalar := candidate.CreateReplacementCandidate(node)
		results.PushBack(scalar)
	}
	return context.ChildContext(results), nil
}
