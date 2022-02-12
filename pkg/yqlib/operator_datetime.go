package yqlib

import (
	"container/list"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func getStringParamter(parameterName string, d *dataTreeNavigator, context Context, expressionNode *ExpressionNode) (string, error) {
	result, err := d.GetMatchingNodes(context.ReadOnlyClone(), expressionNode)

	if err != nil {
		return "", err
	} else if result.MatchingNodes.Len() == 0 {
		return "", fmt.Errorf("could not find %v for format_time", parameterName)
	}

	return result.MatchingNodes.Front().Value.(*CandidateNode).Node.Value, nil
}

func getLayoutAndOtherParam(other string, d *dataTreeNavigator, context Context, expressionNode *ExpressionNode) (string, string, error) {
	layout := time.RFC3339 // reasonable default layout of "2006-01-02T15:04:05Z07:00"
	formatString := ""
	var err error

	if expressionNode.RHS.Operation.OperationType == blockOpType {
		layout, err = getStringParamter("layout", d, context.ReadOnlyClone(), expressionNode.RHS.LHS)

		if err != nil {
			return "", "", err
		}

		// we must have been given a layout and other.
		if formatString, err = getStringParamter("other", d, context.ReadOnlyClone(), expressionNode.RHS.RHS); err != nil {
			return "", "", err
		}

	} else {
		if formatString, err = getStringParamter("other", d, context.ReadOnlyClone(), expressionNode.RHS); err != nil {
			return "", "", err
		}

	}
	return layout, formatString, nil

}

// for unit tests
var Now = time.Now

func nowOp(d *dataTreeNavigator, context Context, expressionNode *ExpressionNode) (Context, error) {

	node := &yaml.Node{
		Tag:   "!!timestamp",
		Kind:  yaml.ScalarNode,
		Value: Now().Format(time.RFC3339),
	}

	return context.SingleChildContext(&CandidateNode{Node: node}), nil

}

func formatDateTime(d *dataTreeNavigator, context Context, expressionNode *ExpressionNode) (Context, error) {
	layout, format, err := getLayoutAndOtherParam("format", d, context, expressionNode)
	decoder := NewYamlDecoder()

	if err != nil {
		return Context{}, err
	}
	var results = list.New()

	for el := context.MatchingNodes.Front(); el != nil; el = el.Next() {
		candidate := el.Value.(*CandidateNode)

		parsedTime, err := time.Parse(layout, candidate.Node.Value)
		if err != nil {
			return Context{}, fmt.Errorf("could not parse datetime of [%v] using layout [%v]: %w", candidate.GetNicePath(), layout, err)
		}
		formattedTimeStr := parsedTime.Format(format)
		decoder.Init(strings.NewReader(formattedTimeStr))
		var dataBucket yaml.Node
		errorReading := decoder.Decode(&dataBucket)
		var node *yaml.Node
		if errorReading != nil {
			log.Debugf("could not parse %v - lets just leave it as a string", formattedTimeStr)
			node = &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: formattedTimeStr,
			}
		} else {
			node = unwrapDoc(&dataBucket)
		}

		results.PushBack(candidate.CreateReplacement(node))
	}

	return context.ChildContext(results), nil
}

func tzOp(d *dataTreeNavigator, context Context, expressionNode *ExpressionNode) (Context, error) {
	layout, timezoneStr, err := getLayoutAndOtherParam("timezone", d, context, expressionNode)

	if err != nil {
		return Context{}, err
	}
	var results = list.New()

	timezone, err := time.LoadLocation(timezoneStr)
	if err != nil {
		return Context{}, fmt.Errorf("could not load tz [%v]: %w", timezoneStr, err)
	}

	for el := context.MatchingNodes.Front(); el != nil; el = el.Next() {
		candidate := el.Value.(*CandidateNode)

		parsedTime, err := time.Parse(layout, candidate.Node.Value)
		if err != nil {
			return Context{}, fmt.Errorf("could not parse datetime of [%v] using layout [%v]: %w", candidate.GetNicePath(), layout, err)
		}
		tzTime := parsedTime.In(timezone)

		node := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   candidate.Node.Tag,
			Value: tzTime.Format(layout),
		}

		results.PushBack(candidate.CreateReplacement(node))
	}

	return context.ChildContext(results), nil
}
