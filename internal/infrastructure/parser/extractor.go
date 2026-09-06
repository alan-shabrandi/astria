package parser

import (
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// StructuralNode represents an extracted piece of code like a Struct, Interface, or Class.
type StructuralNode struct {
	Type       string // "Struct", "Interface", "Class"
	Name       string // Name of the structure
	DocComment string
	StartPoint sitter.Point // Line and column where it starts
	EndPoint   sitter.Point // Line and column where it ends
	Content    string       // The actual raw code block
}

// FunctionNode represents an extracted function or method, including its signature.
type FunctionNode struct {
	IsMethod   bool
	Receiver   string // Only for methods (e.g., "(e *Engine)")
	Name       string
	DocComment string
	Parameters string
	Returns    string
	StartPoint sitter.Point
	EndPoint   sitter.Point
	Content    string // The complete raw code of the function/method
}

// Extractor is responsible for running Tree-sitter queries on the AST.
type Extractor struct {
	registry *LanguageRegistry
}

func NewExtractor(registry *LanguageRegistry) *Extractor {
	return &Extractor{
		registry: registry,
	}
}

// ExtractStructures runs language-specific queries to find Structs, Interfaces, and Classes.
func (e *Extractor) ExtractStructures(tree *sitter.Tree, content []byte, extension string) ([]StructuralNode, error) {
	lang, err := e.registry.GetGrammar(extension)
	if err != nil {
		return nil, fmt.Errorf("unsupported extension for extraction: %w", err)
	}

	queryStr := e.getStructureQuery(extension)
	if queryStr == "" {
		return nil, nil // No structural query defined for this language yet
	}

	query, err := sitter.NewQuery([]byte(queryStr), lang)
	if err != nil {
		return nil, fmt.Errorf("failed to compile tree-sitter query: %w", err)
	}
	defer query.Close()

	cursor := sitter.NewQueryCursor()
	defer cursor.Close()

	cursor.Exec(query, tree.RootNode())

	var nodes []StructuralNode

	for {
		match, ok := cursor.NextMatch()
		if !ok {
			break
		}

		var node StructuralNode
		matchType := "Unknown"

		for _, capture := range match.Captures {
			captureName := query.CaptureNameForId(capture.Index)
			capturedNode := capture.Node

			switch captureName {
			case "struct.name", "interface.name", "class.name":
				node.Name = capturedNode.Content(content)
				if captureName == "struct.name" {
					matchType = "Struct"
				} else if captureName == "interface.name" {
					matchType = "Interface"
				} else {
					matchType = "Class"
				}
			case "struct.decl", "interface.decl", "class.decl":
				node.Type = matchType
				node.StartPoint = capturedNode.StartPoint()
				node.EndPoint = capturedNode.EndPoint()
				node.Content = capturedNode.Content(content)
				node.DocComment = e.extractDocComment(capturedNode, content, extension)
			}
		}

		if node.Name != "" && node.Content != "" {
			node.Type = matchType
			nodes = append(nodes, node)
		}
	}

	return nodes, nil
}

// ExtractFunctions runs language-specific queries to find Functions and Methods.
func (e *Extractor) ExtractFunctions(tree *sitter.Tree, content []byte, extension string) ([]FunctionNode, error) {
	lang, err := e.registry.GetGrammar(extension)
	if err != nil {
		return nil, fmt.Errorf("unsupported extension for function extraction: %w", err)
	}

	queryStr := e.getFunctionQuery(extension)
	if queryStr == "" {
		return nil, nil
	}

	query, err := sitter.NewQuery([]byte(queryStr), lang)
	if err != nil {
		return nil, fmt.Errorf("failed to compile function tree-sitter query: %w", err)
	}
	defer query.Close()

	cursor := sitter.NewQueryCursor()
	defer cursor.Close()

	cursor.Exec(query, tree.RootNode())

	var nodes []FunctionNode
	var currentNode FunctionNode

	for {
		match, ok := cursor.NextMatch()
		if !ok {
			break
		}

		currentNode = FunctionNode{}

		for _, capture := range match.Captures {
			captureName := query.CaptureNameForId(capture.Index)
			capturedNode := capture.Node
			capturedText := capturedNode.Content(content)

			switch captureName {
			case "func.name", "method.name":
				currentNode.Name = capturedText
				if captureName == "method.name" {
					currentNode.IsMethod = true
				}
			case "method.receiver":
				currentNode.Receiver = capturedText
			case "func.params", "method.params":
				currentNode.Parameters = capturedText
			case "func.returns", "method.returns":
				currentNode.Returns = capturedText
			case "func.decl", "method.decl":
				currentNode.StartPoint = capturedNode.StartPoint()
				currentNode.EndPoint = capturedNode.EndPoint()
				currentNode.Content = capturedText
				currentNode.DocComment = e.extractDocComment(capturedNode, content, extension)
			}
		}

		if currentNode.Name != "" && currentNode.Content != "" {
			nodes = append(nodes, currentNode)
		}
	}

	return nodes, nil
}

// getStructureQuery returns the LISP-like query for structural extraction based on the language.
func (e *Extractor) getStructureQuery(extension string) string {
	switch extension {
	case ".go":
		return `
		(type_declaration
			(type_spec
				name: (type_identifier) @struct.name
				type: (struct_type)
			)
		) @struct.decl

		(type_declaration
			(type_spec
				name: (type_identifier) @interface.name
				type: (interface_type)
			)
		) @interface.decl
		`
	case ".py":
		return `
		(class_definition
			name: (identifier) @class.name
		) @class.decl
		`
	default:
		return ""
	}
}

// getFunctionQuery returns the query for extracting functions, methods, and their signatures.
func (e *Extractor) getFunctionQuery(extension string) string {
	switch extension {
	case ".go":
		return `
		(function_declaration
			name: (identifier) @func.name
			parameters: (parameter_list) @func.params
			result: (_)? @func.returns
		) @func.decl

		(method_declaration
			receiver: (parameter_list) @method.receiver
			name: (field_identifier) @method.name
			parameters: (parameter_list) @method.params
			result: (_)? @method.returns
		) @method.decl
		`
	case ".py":
		return `
		(function_definition
			name: (identifier) @func.name
			parameters: (parameters) @func.params
			return_type: (type)? @func.returns
		) @func.decl
		`
	default:
		return ""
	}
}

func (e *Extractor) extractDocComment(node *sitter.Node, content []byte, extension string) string {
	if extension == ".go" {
		var comments []string
		prev := node.PrevNamedSibling()

		for prev != nil && prev.Type() == "comment" {
			comments = append([]string{prev.Content(content)}, comments...)
			prev = prev.PrevNamedSibling()
		}
		return strings.Join(comments, "\n")

	} else if extension == ".py" {
		block := node.ChildByFieldName("body")
		if block != nil && block.NamedChildCount() > 0 {
			firstStmt := block.NamedChild(0)
			if firstStmt.Type() == "expression_statement" {
				strNode := firstStmt.NamedChild(0)
				if strNode != nil && strNode.Type() == "string" {
					return strNode.Content(content)
				}
			}
		}
	}

	return ""
}
