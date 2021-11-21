package yqlib

import (
	"bytes"
	"container/list"
	"fmt"
	"io"
	"os"

	parser "github.com/goccy/go-yaml/parser"
	yaml "gopkg.in/yaml.v3"
)

// A yaml expression evaluator that runs the expression multiple times for each given yaml document.
// Uses less memory than loading all documents and running the expression once, but this cannot process
// cross document expressions.
type StreamEvaluator interface {
	Evaluate(filename string, reader io.Reader, node *ExpressionNode, printer Printer, leadingContent string) (uint, error)
	EvaluateFiles(expression string, filenames []string, printer Printer, leadingContentPreProcessing bool) error
	EvaluateNew(expression string, printer Printer, leadingContent string) error
}

type streamEvaluator struct {
	treeNavigator DataTreeNavigator
	treeCreator   ExpressionParser
	fileIndex     int
}

func NewStreamEvaluator() StreamEvaluator {
	return &streamEvaluator{treeNavigator: NewDataTreeNavigator(), treeCreator: NewExpressionParser()}
}

func (s *streamEvaluator) EvaluateNew(expression string, printer Printer, leadingContent string) error {
	node, err := s.treeCreator.ParseExpression(expression)
	if err != nil {
		return err
	}
	candidateNode := &CandidateNode{
		Document:       0,
		Filename:       "",
		Node:           &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Tag: "!!null", Kind: yaml.ScalarNode}}},
		FileIndex:      0,
		LeadingContent: leadingContent,
	}
	inputList := list.New()
	inputList.PushBack(candidateNode)

	result, errorParsing := s.treeNavigator.GetMatchingNodes(Context{MatchingNodes: inputList}, node)
	if errorParsing != nil {
		return errorParsing
	}
	return printer.PrintResults(result.MatchingNodes)
}

func (s *streamEvaluator) EvaluateFiles(expression string, filenames []string, printer Printer, leadingContentPreProcessing bool) error {
	var totalProcessDocs uint = 0
	node, err := s.treeCreator.ParseExpression(expression)
	if err != nil {
		return err
	}

	var firstFileLeadingContent string

	for index, filename := range filenames {
		reader, leadingContent, err := readStream(filename, leadingContentPreProcessing)

		if index == 0 {
			firstFileLeadingContent = leadingContent
		}

		if err != nil {
			return err
		}
		processedDocs, err := s.Evaluate(filename, reader, node, printer, leadingContent)
		if err != nil {
			return err
		}
		totalProcessDocs = totalProcessDocs + processedDocs

		switch reader := reader.(type) {
		case *os.File:
			safelyCloseFile(reader)
		}
	}

	if totalProcessDocs == 0 {
		return s.EvaluateNew(expression, printer, firstFileLeadingContent)
	}

	return nil
}

func (s *streamEvaluator) Evaluate(filename string, reader io.Reader, node *ExpressionNode, printer Printer, leadingContent string) (uint, error) {

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		return 0, fmt.Errorf("failed to copy from reader: %w", err)
	}

	f, err := parser.ParseBytes(buf.Bytes(), parser.ParseComments)

	if err != nil {
		return 0, fmt.Errorf("failed to decode: %w", err)
	}

	for index, doc := range f.Docs {
		currentIndex := uint(index)
		log.Debugf("doc: <%v>", doc.String())
		log.Debugf("start: <%v>", doc.Start)
		log.Debugf("end: <%v>", doc.End)
		log.Debugf("body: <%v>", doc.Body.String())
		log.Debugf("comment: <%v>", doc.Body.GetComment())

		candidateNode := &CandidateNode{
			Document:  currentIndex,
			Filename:  f.Name,
			Node:      &doc.Body, // should this be doc??
			FileIndex: s.fileIndex,
		}

		result, errorParsing := s.treeNavigator.GetMatchingNodes(Context{MatchingNodes: candidateNode.AsList()}, node)
		if errorParsing != nil {
			return currentIndex, errorParsing
		}
		err := printer.PrintResults(result.MatchingNodes)
		if err != nil {
			return currentIndex, err
		}
	}
	return uint(len(f.Docs)), nil

	// for {
	// 	var dataBucket ast.Node
	// 	errorReading := decoder.Decode(&dataBucket)

	// 	if errorReading == io.EOF {
	// 		s.fileIndex = s.fileIndex + 1
	// 		return currentIndex, nil
	// 	} else if errorReading != nil {
	// 		return currentIndex, errorReading
	// 	}
	// 	log.Debugf("goccy! %v", dataBucket.Type().String())
	// 	log.Debugf("goccy! %v", dataBucket.String())
	// 	dataBucket.MarshalYAML()

	// candidateNode := &CandidateNode{
	// 	Document:  currentIndex,
	// 	Filename:  filename,
	// 	Node:      &dataBucket,
	// 	FileIndex: s.fileIndex,
	// }
	// if currentIndex == 0 {
	// 	candidateNode.LeadingContent = leadingContent
	// }
	// inputList := list.New()
	// inputList.PushBack(candidateNode)

	// result, errorParsing := s.treeNavigator.GetMatchingNodes(Context{MatchingNodes: inputList}, node)
	// if errorParsing != nil {
	// 	return currentIndex, errorParsing
	// }
	// err := printer.PrintResults(result.MatchingNodes)

	// if err != nil {
	// 	return currentIndex, err
	// }
	// currentIndex = currentIndex + 1
	// }
}
