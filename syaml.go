package syaml

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

func Keys(path string) []string {
	return strings.Split(path, ".")
}

func FileApply(file string, op *Traversal) error {
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}
	return BytesApply(bytes, op)
}

func BytesApply(data []byte, op *Traversal) error {
	var node yaml.Node

	r := bytes.NewReader(data)

	dec := yaml.NewDecoder(r)

	var errs []error

	var nodeCnt int

	for {
		if err := dec.Decode(&node); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}

		nodeCnt++

		if err := Apply(&node, op, nil); err != nil {
			errs = append(errs, err)
		}

		if nodeCnt > 1 {
			fmt.Fprintf(os.Stdout, "---\n")
		}

		DumpYaml(&node)
	}

	Debugf("nodeCnt=%d, errs=%d\n", nodeCnt, len(errs))

	if len(errs) == nodeCnt {
		return errs[0]
	}

	return nil
}

type Label string

// Traversal is a radix tree of yaml path to function that visits the specific value in the document
type Traversal struct {
	Label Label

	Visit func(*yaml.Node) error

	Children map[Label]*Traversal
}

func (t *Traversal) Add(p []Label, f func(*yaml.Node) error) {
	if t.Children == nil {
		t.Children = map[Label]*Traversal{}
	}

	child, ok := t.Children[p[0]]
	if !ok {
		child = &Traversal{
			Label: p[0],
		}

		t.Children[p[0]] = child
	}

	if len(p) == 1 {
		child.Visit = f
	} else {
		child.Add(p[1:], f)
	}
}

func (t *Traversal) InPlaceMerge(other *Traversal) *Traversal {
	othersChildren := map[Label]*Traversal{}

	for k, v := range other.Children {
		othersChildren[k] = v
	}

	for label, a := range t.Children {
		b, ok := othersChildren[label]
		if ok {
			delete(othersChildren, label)

			a.InPlaceMerge(b)
		}
	}

	for label, a := range othersChildren {
		if t.Children == nil {
			t.Children = map[Label]*Traversal{}
		}

		t.Children[label] = a
	}

	return t
}

func (n *Traversal) IsLeaf() bool {
	return n.Children == nil || len(n.Children) == 0
}

func Set(pathComponents []string, value interface{}) *Traversal {
	entries := []struct {
		pathComponents []string
		f              func(valNode *yaml.Node) error
	}{
		{
			pathComponents: pathComponents,
			f: func(valNode *yaml.Node) error {
				switch valNode.Kind {
				case yaml.ScalarNode:
					valNode.Value = fmt.Sprintf("%v", value)

					switch value.(type) {
					case string:
						valNode.Tag = "!!str"
						valNode.Style = yaml.DoubleQuotedStyle
					case int:
						valNode.Tag = "!!int"
					default:
						panic(fmt.Errorf("unexpected type of value to set: %v(%T)", value, value))
					}
					return nil
				case yaml.MappingNode:
					valNode.Value = fmt.Sprintf("%v", value)
					valNode.Kind = yaml.ScalarNode
					// Avoid turning something like `bar: !!map {"foo":"bar"}` into `bar: !!map 1`
					// (valNode.Tag is `!!map` in the above example
					valNode.Tag = ""
					valNode.Content = nil
					return nil
				default:
					panic(fmt.Errorf("unexpected condition1 "))
				}

				return nil
			},
		},
	}

	patch := &Traversal{}

	for _, ent := range entries {
		labels := make([]Label, len(ent.pathComponents))

		for i, c := range ent.pathComponents {
			labels[i] = Label(c)
		}

		patch.Add(labels, ent.f)
	}

	return patch
}

type Condition struct {
	Traversal *Traversal

	Result func() bool
}

func And(conds ...*Condition) *Condition {
	var aggregated Traversal

	for _, c := range conds {
		aggregated.InPlaceMerge(c.Traversal)
	}

	return &Condition{
		Traversal: &aggregated,
		Result: func() bool {
			agg := true
			for _, c := range conds {
				agg = agg && c.Result()
			}
			return agg
		},
	}
}

func Or(conds ...*Condition) *Condition {
	var aggregated Traversal

	for _, c := range conds {
		aggregated.InPlaceMerge(c.Traversal)
	}

	return &Condition{
		Traversal: &aggregated,
		Result: func() bool {
			agg := false
			for _, c := range conds {
				agg = agg || c.Result()
			}
			return agg
		},
	}
}

func Eq(pathComponents []string, value interface{}) *Condition {
	var result bool

	entries := []struct {
		pathComponents []string
		f              func(valNode *yaml.Node) error
	}{
		{
			pathComponents: pathComponents,
			f: func(valNode *yaml.Node) error {
				result = valNode.Value == value

				return nil
			},
		},
	}

	patch := &Traversal{}

	for _, ent := range entries {
		labels := make([]Label, len(ent.pathComponents))

		for i, c := range ent.pathComponents {
			labels[i] = Label(c)
		}

		patch.Add(labels, ent.f)
	}

	return &Condition{Traversal: patch, Result: func() bool {
		return result
	}}
}

func Match(node *yaml.Node, cond *Condition) (bool, error) {
	if cond == nil {
		return true, nil
	}

	_, err := Traverse(node, cond.Traversal, false)
	if err != nil {
		return false, nil
	}

	return cond.Result(), nil
}

func Apply(node *yaml.Node, op *Traversal, cond *Condition) error {
	if cond != nil {
		_, err := Traverse(node, cond.Traversal, false)
		if err != nil {
			return err
		}

		if !cond.Result() {
			Errorf("Document didnt match the condition\n")

			return nil
		}
	}

	_, err := Traverse(node, op, true)

	return err
}

type ValueNotFoundError struct {
}

func (e *ValueNotFoundError) Error() string {
	return "value not found for path(s)"
}

func Traverse(node *yaml.Node, tree *Traversal, createMissing bool) (*int, error) {
	Debugf("node = %v, key = %s\n", node.Content, tree.Label)

	var found int

	switch node.Kind {
	case yaml.DocumentNode:
		// doc
		for _, v := range node.Content {
			n, err := Traverse(v, tree, createMissing)
			if err != nil && !errors.Is(err, &ValueNotFoundError{}) {
				return nil, err
			}

			if n != nil {
				found += *n
			}
		}
	case yaml.MappingNode:
		processed := map[Label]bool{}

		// map
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valNode := node.Content[i+1]

			if keyNode.Kind == yaml.ScalarNode {
				key := keyNode.Value

				label := Label(key)

				child, ok := tree.Children[label]
				if ok {
					if child.IsLeaf() {
						if child.Visit != nil {
							if err := child.Visit(valNode); err != nil {
								return nil, err
							}
						}

						found++

						processed[label] = true
					} else if valNode.Kind == yaml.MappingNode {
						n, err := Traverse(valNode, child, createMissing)
						if err != nil {
							return nil, err
						}

						found += *n

						processed[label] = true
					} else {
						panic(fmt.Errorf("unexpected condition %q: %v", key, valNode.Kind))
					}
				}
			}
		}

		if createMissing {
			unprocessed := map[Label]*Traversal{}

			for k, v := range tree.Children {
				if !processed[k] {
					unprocessed[k] = v
				}
			}

			for _, v := range unprocessed {
				keyNode, valNode, err := treeToYamlMapping(v)
				if err != nil {
					return nil, err
				}

				node.Content = append(node.Content, keyNode, valNode)
			}
		}
	}

	if found == 0 {
		return nil, &ValueNotFoundError{}
	}

	return &found, nil
}

func treeToYamlMapping(v *Traversal) (*yaml.Node, *yaml.Node, error) {
	k := string(v.Label)

	keyNode := &yaml.Node{
		Kind:        yaml.ScalarNode,
		Style:       yaml.DoubleQuotedStyle,
		Tag:         "!!str",
		Value:       k,
		Anchor:      "",
		Alias:       nil,
		Content:     nil,
		HeadComment: "",
		LineComment: "",
		FootComment: "",
		Line:        0,
		Column:      0,
	}

	var valNode *yaml.Node

	if !v.IsLeaf() {
		valNode = &yaml.Node{
			Kind:        yaml.MappingNode,
			Style:       0,
			Tag:         "!!map",
			Value:       "",
			Anchor:      "",
			Alias:       nil,
			Content:     []*yaml.Node{},
			HeadComment: "",
			LineComment: "",
			FootComment: "",
			Line:        0,
			Column:      0,
		}

		for _, child := range v.Children {
			childKey, childValue, err := treeToYamlMapping(child)
			if err != nil {
				return nil, nil, err
			}

			valNode.Content = append(valNode.Content, childKey, childValue)
		}
	} else {
		valNode = &yaml.Node{
			Kind:        yaml.ScalarNode,
			Style:       0,
			Tag:         "!!str",
			Value:       "<no value>",
			Anchor:      "",
			Alias:       nil,
			Content:     nil,
			HeadComment: "",
			LineComment: "",
			FootComment: "",
			Line:        0,
			Column:      0,
		}

		if err := v.Visit(valNode); err != nil {
			return nil, nil, err
		}
	}

	return keyNode, valNode, nil
}

func DumpYaml(node *yaml.Node) {
	if DebugEnabled() {
		buf := bytes.Buffer{}

		enc := json.NewEncoder(&buf)
		enc.SetIndent("", "  ")

		jsonErr := enc.Encode(node)
		if jsonErr == nil {
			fmt.Fprintf(os.Stderr, "%s\n", buf.String())
		}
	}

	enc := yaml.NewEncoder(os.Stdout)
	enc.SetIndent(2)
	if err := enc.Encode(node); err != nil {
		panic(fmt.Errorf("%v", err))
	}
}

func DebugEnabled() bool {
	return os.Getenv("DEBUG") != ""
}

func Debugf(f string, args ...interface{}) {
	if DebugEnabled() {
		Errorf(f, args...)
	}
}

func Errorf(f string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, f, args...)
}
