package yqlib

import (
	"container/list"

	"github.com/goccy/go-yaml/ast"
)

func collectOperator(d *dataTreeNavigator, context Context, expressionNode *ExpressionNode) (Context, error) {
	log.Debugf("-- collectOperation")

	if context.MatchingNodes.Len() == 0 {
		candidate := &CandidateNode{Node: &ast.SequenceNode{}}
		return context.SingleChildContext(candidate), nil
	}

	var results = list.New()

	node := &ast.SequenceNode{}
	var collectC *CandidateNode
	// if context.MatchingNodes.Front() != nil {
	// 	collectC = context.MatchingNodes.Front().Value.(*CandidateNode).CreateChild(nil, node)
	// 	if len(collectC.Path) > 0 {
	// 		collectC.Path = collectC.Path[:len(collectC.Path)-1]
	// 	}
	// } else {
	collectC = &CandidateNode{Node: node}
	// }

	for el := context.MatchingNodes.Front(); el != nil; el = el.Next() {
		candidate := el.Value.(*CandidateNode)
		log.Debugf("Collecting %v", NodeToString(candidate))
		node.Values = append(node.Values, unwrapDoc(candidate.Node))
	}

	results.PushBack(collectC)

	return context.ChildContext(results), nil
}
