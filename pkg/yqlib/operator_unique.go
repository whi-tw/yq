package yqlib

import (
	"container/list"
	"fmt"

	"github.com/elliotchance/orderedmap"
	"github.com/goccy/go-yaml/ast"
)

func unique(d *dataTreeNavigator, context Context, expressionNode *ExpressionNode) (Context, error) {
	selfExpression := &ExpressionNode{Operation: &Operation{OperationType: selfReferenceOpType}}
	uniqueByExpression := &ExpressionNode{Operation: &Operation{OperationType: uniqueByOpType}, Rhs: selfExpression}
	return uniqueBy(d, context, uniqueByExpression)

}

func uniqueBy(d *dataTreeNavigator, context Context, expressionNode *ExpressionNode) (Context, error) {

	log.Debugf("-- uniqueBy Operator")
	var results = list.New()

	for el := context.MatchingNodes.Front(); el != nil; el = el.Next() {
		candidate := el.Value.(*CandidateNode)
		candidateNode := unwrapDoc(candidate.Node)

		if candidateNode.Type() != ast.SequenceType {
			return Context{}, fmt.Errorf("Only arrays are supported for unique")
		}

		var newMatches = orderedmap.NewOrderedMap()
		for _, node := range candidateNode.(*ast.SequenceNode).Values {
			child := &CandidateNode{Node: node}
			rhs, err := d.GetMatchingNodes(context.SingleReadonlyChildContext(child), expressionNode.Rhs)

			if err != nil {
				return Context{}, err
			}

			keyValue := "null"

			if rhs.MatchingNodes.Len() > 0 {
				first := rhs.MatchingNodes.Front()
				keyCandidate := first.Value.(*CandidateNode)
				keyValue = keyCandidate.Node.String()
			}

			_, exists := newMatches.Get(keyValue)

			if !exists {
				newMatches.Set(keyValue, child.Node)
			}
		}
		resultNode := &ast.SequenceNode{}
		for el := newMatches.Front(); el != nil; el = el.Next() {
			resultNode.Values = append(resultNode.Values, el.Value.(ast.Node))
		}

		results.PushBack(candidate.CreateReplacementCandidate(resultNode))
	}

	return context.ChildContext(results), nil

}
