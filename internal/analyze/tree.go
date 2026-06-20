package analyze

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

const maxPreviewRunes = 160

// TreeNode describes one recursively measured JSON property or array element.
type TreeNode struct {
	Name     string     `json:"name"`
	Label    string     `json:"label,omitempty"`
	Kind     string     `json:"kind"`
	Preview  string     `json:"preview,omitempty"`
	Children []TreeNode `json:"children,omitempty"`
	Bytes    int        `json:"bytes"`
}

// Tree is the recursively measured representation used by the web report.
type Tree struct {
	Root            TreeNode `json:"root"`
	CompressedBytes int      `json:"compressed_bytes"`
}

// BuildTreeValidated recursively measures release JSON that the caller has
// already validated. Callers must not use this function with untrusted or
// unvalidated input.
func BuildTreeValidated(releaseJSON []byte) (Tree, error) {
	start := skipWhitespace(releaseJSON, 0)
	if start >= len(releaseJSON) || releaseJSON[start] != '{' {
		return Tree{}, errTopLevelObject
	}

	root, err := buildTreeNode("root", releaseJSON[start:], len(releaseJSON))
	if err != nil {
		return Tree{}, err
	}

	return Tree{Root: root}, nil
}

func buildTreeNode(name string, value []byte, size int) (TreeNode, error) {
	start := skipWhitespace(value, 0)
	if start >= len(value) {
		return TreeNode{}, errObjectTerminated
	}

	node := TreeNode{Name: name, Bytes: size}

	var err error

	switch value[start] {
	case '{':
		node.Kind = "object"
		node.Children, err = buildObjectChildren(value[start:])
	case '[':
		node.Kind = "array"
		node.Children, err = buildArrayChildren(value[start:])
	case '"':
		node.Kind = "string"
		node.Preview, err = stringPreview(value[start:])
	case 't', 'f':
		node.Kind = "boolean"
		node.Preview = string(value[start:])
	case 'n':
		node.Kind = "null"
		node.Preview = "null"
	default:
		node.Kind = "number"
		node.Preview = string(value[start:])
	}

	return node, err
}

func buildObjectChildren(data []byte) ([]TreeNode, error) {
	cursor := 1
	children := make([]TreeNode, 0)

	for {
		measured, next, done, err := measureProperty(data, cursor)
		if err != nil {
			return nil, err
		}

		if measured.Bytes == 0 {
			return children, nil
		}

		child, err := buildTreeNode(
			measured.Name,
			data[measured.valueStart:measured.valueEnd],
			measured.Bytes,
		)
		if err != nil {
			return nil, fmt.Errorf("measure property %q: %w", measured.Name, err)
		}

		children = append(children, child)
		if done {
			return children, nil
		}

		cursor = next
	}
}

func buildArrayChildren(data []byte) ([]TreeNode, error) {
	cursor := 1
	children := make([]TreeNode, 0)

	for index := 0; ; index++ {
		child, next, done, err := measureArrayElement(data, cursor, index)
		if err != nil {
			return nil, err
		}

		if child.Bytes == 0 {
			return children, nil
		}

		children = append(children, child)
		if done {
			return children, nil
		}

		cursor = next
	}
}

func measureArrayElement(data []byte, elementStart, index int) (TreeNode, int, bool, error) {
	valueStart := skipWhitespace(data, elementStart)
	if valueStart >= len(data) {
		return TreeNode{}, 0, false, errObjectTerminated
	}

	if data[valueStart] == ']' {
		return TreeNode{}, valueStart, true, nil
	}

	decoder := json.NewDecoder(bytes.NewReader(data[valueStart:]))

	var value json.RawMessage

	err := decoder.Decode(&value)
	if err != nil {
		return TreeNode{}, 0, false, fmt.Errorf("decode array element %d: %w", index, err)
	}

	valueEnd := valueStart + int(decoder.InputOffset())

	cursor := skipWhitespace(data, valueEnd)
	if cursor >= len(data) {
		return TreeNode{}, 0, false, errObjectTerminated
	}

	done := data[cursor] == ']'
	if !done && data[cursor] != ',' {
		return TreeNode{}, 0, false, errPropertyEnd
	}

	if !done {
		cursor = skipWhitespace(data, cursor+1)
	}

	child, err := buildTreeNode(
		strconv.Itoa(index),
		data[valueStart:valueEnd],
		cursor-elementStart,
	)
	if err != nil {
		return TreeNode{}, 0, false, fmt.Errorf("measure array element %d: %w", index, err)
	}

	child.Label = arrayElementLabel(child)

	return child, cursor, done, nil
}

func arrayElementLabel(node TreeNode) string {
	if node.Kind != "object" {
		return ""
	}

	for _, property := range node.Children {
		if property.Name == "name" && property.Kind == "string" {
			return property.Preview
		}
	}

	return ""
}

func stringPreview(data []byte) (string, error) {
	var value string

	err := json.Unmarshal(data, &value)
	if err != nil {
		return "", fmt.Errorf("decode string preview: %w", err)
	}

	runes := []rune(value)
	if len(runes) <= maxPreviewRunes {
		return value, nil
	}

	return string(runes[:maxPreviewRunes]) + "…", nil
}
