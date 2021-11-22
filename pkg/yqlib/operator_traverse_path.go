package yqlib

import (
	"container/list"
	"fmt"
	"reflect"

	"github.com/elliotchance/orderedmap"
	"github.com/goccy/go-yaml/ast"
)

type traversePreferences struct {
	DontFollowAlias      bool
	IncludeMapKeys       bool
	DontAutoCreate       bool // by default, we automatically create entries on the fly.
	DontIncludeMapValues bool
	OptionalTraverse     bool // e.g. .adf?
}

func splat(d *dataTreeNavigator, context Context, prefs traversePreferences) (Context, error) {
	return traverseNodesWithArrayIndices(context, make([]ast.Node, 0), prefs)
}

func traversePathOperator(d *dataTreeNavigator, context Context, expressionNode *ExpressionNode) (Context, error) {
	log.Debugf("-- traversePathOperator")
	var matches = list.New()

	for el := context.MatchingNodes.Front(); el != nil; el = el.Next() {
		newNodes, err := traverse(d, context, el.Value.(*CandidateNode), expressionNode.Operation)
		if err != nil {
			return Context{}, err
		}
		matches.PushBackList(newNodes)
	}

	return context.ChildContext(matches), nil
}

func traverse(d *dataTreeNavigator, context Context, matchingNode *CandidateNode, operation *Operation) (*list.List, error) {
	log.Debug("Traversing %v", NodeToString(matchingNode))
	value := matchingNode.Node

	if value.Type() == ast.NullType && operation.Value != "[]" {
		log.Debugf("Guessing kind")
		// we must ahve added this automatically, lets guess what it should be now
		switch operation.Value.(type) {
		case int, int64:
			log.Debugf("probably an array")
			matchingNode.ReplaceWith(&ast.SequenceNode{})
		default:
			log.Debugf("probably a map")
			matchingNode.ReplaceWith(&ast.MappingNode{})
		}
	}

	switch typedValue := value.(type) {
	case *ast.MappingNode:
		log.Debug("its a map with %v entries", len(typedValue.Values))
		return traverseMap(context, matchingNode, operation.StringValue, operation.Preferences.(traversePreferences), false)

	case *ast.SequenceNode:
		log.Debug("its a sequence of %v things!", len(typedValue.Values))
		return traverseArray(matchingNode, operation, operation.Preferences.(traversePreferences))

	case *ast.AliasNode:
		log.Debug("its an alias!")
		matchingNode.Node = typedValue.Value
		return traverse(d, context, matchingNode, operation)
	case *ast.DocumentNode:
		log.Debug("digging into doc node")
		matchingNode.Node = typedValue.Body
		return traverse(d, context, matchingNode, operation)
	default:
		return list.New(), nil
	}
}

func traverseArrayOperator(d *dataTreeNavigator, context Context, expressionNode *ExpressionNode) (Context, error) {

	//lhs may update the variable context, we should pass that into the RHS
	// BUT we still return the original context back (see jq)
	// https://stedolan.github.io/jq/manual/#Variable/SymbolicBindingOperator:...as$identifier|...

	lhs, err := d.GetMatchingNodes(context, expressionNode.Lhs)
	if err != nil {
		return Context{}, err
	}

	// rhs is a collect expression that will yield indexes to retreive of the arrays

	rhs, err := d.GetMatchingNodes(context.ReadOnlyClone(), expressionNode.Rhs)

	if err != nil {
		return Context{}, err
	}
	prefs := traversePreferences{}

	if expressionNode.Rhs.Rhs != nil && expressionNode.Rhs.Rhs.Operation.Preferences != nil {
		prefs = expressionNode.Rhs.Rhs.Operation.Preferences.(traversePreferences)
	}
	var indicesToTraverse = rhs.MatchingNodes.Front().Value.(*CandidateNode).Node.(*ast.SequenceNode).Values

	//now we traverse the result of the lhs against the indices we found
	result, err := traverseNodesWithArrayIndices(lhs, indicesToTraverse, prefs)
	if err != nil {
		return Context{}, err
	}
	return context.ChildContext(result.MatchingNodes), nil
}

func traverseNodesWithArrayIndices(context Context, indicesToTraverse []ast.Node, prefs traversePreferences) (Context, error) {
	var matchingNodeMap = list.New()
	for el := context.MatchingNodes.Front(); el != nil; el = el.Next() {
		candidate := el.Value.(*CandidateNode)
		newNodes, err := traverseArrayIndices(context, candidate, indicesToTraverse, prefs)
		if err != nil {
			return Context{}, err
		}
		matchingNodeMap.PushBackList(newNodes)
	}

	return context.ChildContext(matchingNodeMap), nil
}

func traverseArrayIndices(context Context, matchingNode *CandidateNode, indicesToTraverse []ast.Node, prefs traversePreferences) (*list.List, error) { // call this if doc / alias like the other traverse
	node := matchingNode.Node

	if node.Type() == ast.NullType {
		log.Debugf("OperatorArrayTraverse got a null - turning it into an empty array")
		// auto vivification
		node = &ast.SequenceNode{}

		//check that the indices are numeric, if not, then we should create an object
		if len(indicesToTraverse) != 0 && indicesToTraverse[0].Type() != ast.IntegerType {
			node = &ast.MappingNode{}
		}

		err := matchingNode.ReplaceWith(node)
		if err != nil {
			return nil, err
		}
	}

	switch node := node.(type) {
	case *ast.AliasNode:
		matchingNode.Node = node.Value
		return traverseArrayIndices(context, matchingNode, indicesToTraverse, prefs)
	case *ast.SequenceNode:
		return traverseArrayWithIndices(matchingNode, indicesToTraverse, prefs)
	case *ast.MappingNode:
		return traverseMapWithIndices(context, matchingNode, indicesToTraverse, prefs)
	case *ast.DocumentNode:
		matchingNode.Node = node.Body
		return traverseArrayIndices(context, matchingNode, indicesToTraverse, prefs)
	}

	log.Debugf("OperatorArrayTraverse skipping %v as its a %v", matchingNode, node.Type)
	return list.New(), nil
}

func traverseMapWithIndices(context Context, candidate *CandidateNode, indices []ast.Node, prefs traversePreferences) (*list.List, error) {
	if len(indices) == 0 {
		return traverseMap(context, candidate, "", prefs, true)
	}

	var matchingNodeMap = list.New()

	for _, indexNode := range indices {
		log.Debug("traverseMapWithIndices: %v", indexNode)
		newNodes, err := traverseMap(context, candidate, indexNode.(ast.ScalarNode).GetValue(), prefs, false)
		if err != nil {
			return nil, err
		}
		matchingNodeMap.PushBackList(newNodes)
	}

	return matchingNodeMap, nil
}

func traverseArrayWithIndices(candidate *CandidateNode, indices []ast.Node, prefs traversePreferences) (*list.List, error) {
	log.Debug("traverseArrayWithIndices")
	var newMatches = list.New()
	// node := unwrapDoc(candidate.Node)
	node := candidate.Node.(*ast.SequenceNode)
	if len(indices) == 0 {
		log.Debug("splatting")
		for index := 0; index < len(node.Values); index = index + 1 {
			newMatches.PushBack(candidate.CreateArrayChild(index, node.Values[index]))
		}
		return newMatches, nil
	}

	for _, indexNode := range indices {
		log.Debug("traverseArrayWithIndices: '%v'", indexNode)

		if indexNode.Type() != ast.IntegerType && prefs.OptionalTraverse {
			continue
		} else if indexNode.Type() != ast.IntegerType {
			return nil, fmt.Errorf("Cannot index array with '%v'", indexNode)
		}
		index := indexNode.(*ast.IntegerNode).Value.(int)

		contentLength := len(node.Values)
		for contentLength <= index {
			node.Values = append(node.Values, &ast.NullNode{})
			contentLength = len(node.Values)
		}
		indexToUse := index
		if indexToUse < 0 {
			indexToUse = contentLength + indexToUse
		}

		if indexToUse < 0 {
			return nil, fmt.Errorf("Index [%v] out of range, array size is %v", index, contentLength)
		}

		newMatches.PushBack(candidate.CreateArrayChild(index, node.Values[indexToUse]))
	}
	return newMatches, nil
}

func keyMatches(key ast.Node, wantedKey interface{}) bool {
	actualValue := key.(ast.ScalarNode).GetValue()
	actualType := reflect.ValueOf(actualValue)
	wantedType := reflect.ValueOf(wantedKey)

	if actualType == wantedType {
		switch actual := actualValue.(type) {
		case string:
			return matchKey(actual, wantedKey.(string))
		}
	}

	return actualValue == wantedKey
}

func traverseMap(context Context, matchingNode *CandidateNode, key interface{}, prefs traversePreferences, splat bool) (*list.List, error) {
	var newMatches = orderedmap.NewOrderedMap()
	err := doTraverseMap(newMatches, matchingNode, key, prefs, splat)

	if err != nil {
		return nil, err
	}

	if !prefs.DontAutoCreate && !context.DontAutoCreate && newMatches.Len() == 0 {
		//no matches, create one automagically
		valueNode := &ast.NullNode{}
		var keyNode ast.Node
		switch key := key.(type) {
		case string:
			keyNode = &ast.StringNode{Value: key}
		case int, int64, uint, uint64:
			keyNode = &ast.IntegerNode{Value: key}
		case float32:
			keyNode = &ast.FloatNode{Value: float64(key), Precision: 32}
		case float64:
			keyNode = &ast.FloatNode{Value: key, Precision: 64}
		}

		node := matchingNode.Node.(*ast.MappingNode)
		mvn := &ast.MappingValueNode{Key: keyNode, Value: valueNode}
		node.Values = append(node.Values, mvn)

		if prefs.IncludeMapKeys {
			log.Debug("including key")
			candidateNode := matchingNode.CreateMapChild(keyNode, keyNode)
			candidateNode.IsMapKey = true
			newMatches.Set(fmt.Sprintf("keyOf-%v", candidateNode.GetKey()), candidateNode)
		}
		if !prefs.DontIncludeMapValues {
			log.Debug("including value")
			candidateNode := matchingNode.CreateMapChild(keyNode, valueNode)
			newMatches.Set(candidateNode.GetKey(), candidateNode)
		}
	}

	results := list.New()
	i := 0
	for el := newMatches.Front(); el != nil; el = el.Next() {
		results.PushBack(el.Value)
		i++
	}
	return results, nil
}

func doTraverseMap(newMatches *orderedmap.OrderedMap, candidate *CandidateNode, wantedKey interface{}, prefs traversePreferences, splat bool) error {
	// value.Content is a concatenated array of key, value,
	// so keys are in the even indexes, values in odd.
	// merge aliases are defined first, but we only want to traverse them
	// if we don't find a match directly on this node first.

	node := candidate.Node.(*ast.MappingNode)

	for _, mapValueNode := range node.Values {
		key := mapValueNode.Key
		value := mapValueNode.Value

		log.Debug("checking %v", key)
		//skip the 'merge' tag, find a direct match first
		if key.Type() == ast.MergeKeyType && !prefs.DontFollowAlias {
			log.Debug("Merge anchor")
			err := traverseMergeAnchor(newMatches, candidate, value, wantedKey, prefs, splat)
			if err != nil {
				return err
			}
		} else if splat || keyMatches(key, wantedKey) {
			log.Debug("MATCHED")
			if prefs.IncludeMapKeys {
				log.Debug("including key")
				candidateNode := candidate.CreateMapChild(key, key)
				candidateNode.IsMapKey = true
				newMatches.Set(fmt.Sprintf("keyOf-%v", candidateNode.GetKey()), candidateNode)
			}
			if !prefs.DontIncludeMapValues {
				log.Debug("including value")
				candidateNode := candidate.CreateMapChild(key, value)
				newMatches.Set(candidateNode.GetKey(), candidateNode)
			}
		}
	}

	return nil
}

func traverseMergeAnchor(newMatches *orderedmap.OrderedMap, originalCandidate *CandidateNode, value ast.Node, wantedKey interface{}, prefs traversePreferences, splat bool) error {
	switch value := value.(type) {
	case *ast.AliasNode:
		originalCandidate.Node = value.Value
		return doTraverseMap(newMatches, originalCandidate, wantedKey, prefs, splat)
	case *ast.SequenceNode:
		for _, childValue := range value.Values {
			err := traverseMergeAnchor(newMatches, originalCandidate, childValue, wantedKey, prefs, splat)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func traverseArray(candidate *CandidateNode, operation *Operation, prefs traversePreferences) (*list.List, error) {
	log.Debug("operation Value %v", operation.Value)
	var indices []ast.Node
	switch value := operation.Value.(type) {
	case string:
		indices = []ast.Node{&ast.StringNode{Value: value}}
	case int, int64, uint, uint64:
		indices = []ast.Node{&ast.IntegerNode{Value: value}}
	}

	return traverseArrayWithIndices(candidate, indices, prefs)
}
