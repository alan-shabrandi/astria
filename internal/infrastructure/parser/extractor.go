package parser

import (
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
)

// StructuralNode represents an extracted piece of code like a Struct, Interface, or Class.
type StructuralNode struct {
	Type       string       // "Struct", "Interface", "Class"
	Name       string       // Name of the structure
	StartPoint sitter.Point // Line and column where it starts
	EndPoint   sitter.Point // Line and column where it ends
	Content    string       // The actual raw code block
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
			}
		}

		if node.Name != "" && node.Content != "" {
			node.Type = matchType
			nodes = append(nodes, node)
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
