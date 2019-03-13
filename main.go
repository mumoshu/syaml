package main

import (
	"github.com/mumoshu/syaml/forks/github.com/niemeyer/ynext"
	"io/ioutil"
	"fmt"
	"encoding/json"
	"strings"
	"os"
)

func main() {
	args := os.Args[1:]
	if len(args) == 3 {
		if err := FileSet(args[0], keys(args[1]), args[2]); err != nil {
			fmt.Printf("err: %v", err)
		}
	} else if len(args) == 2 {
		bytes, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			panic(err)
		}
		if err := BytesSet(bytes, keys(args[0]), args[1]); err != nil {
			fmt.Printf("err: %v", err)
		}
	} else {
		fmt.Fprintf(os.Stderr, "invalid number of args: you should pass 2 or 3 args, but got: %v\n", args)
		os.Exit(1)
	}
}

func keys(path string) []string {
	return strings.Split(path, ".")
}

func FileSet(file string, keys []string, value interface{}) error {
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}
	return BytesSet(bytes, keys, value)
}

func BytesSet(bytes []byte, keys []string, value interface{}) error {
	var node ynext.Node
	err := ynext.Unmarshal(bytes, &node)
	dumpJson(&node)

	var lastInline string
	linesDec := 0
	deepFirstTraverse(&node, func(n *ynext.Node) {
		if lastInline != "" {
			n.Header = lastInline
			linesDec += 1
		}
		lastInline = n.Inline
		n.Line = n.Line - linesDec
		n.Inline = ""
	})
	dumpJson(&node)

	err = NodeSet(&node, keys, value)

	if err != nil {
		return err
	}

	dumpYaml(&node)

	return nil
}

func deepFirstTraverse(node *ynext.Node, mutate func(node *ynext.Node)) {
	mutate(node)

	if node.Children != nil {
		for _, n := range node.Children {
			deepFirstTraverse(n, mutate)
		}
	}
}

func NodeSet(node *ynext.Node, keys []string, value interface{}) error {
	fmt.Fprintf(os.Stderr, "node = %v, key = %s\n", node.Children, keys[0])
	switch node.Kind {
	case 1:
		// doc
		for _, v := range node.Children {
			err := NodeSet(v, keys, value)
			if err == nil {
				return nil
			}
		}
	case 4:
		// map
		for i := 0; i < len(node.Children); i += 2 {
			c := node.Children[i]
			if c.Kind == 8 && c.Value == keys[0] {
				next := node.Children[i+1]
				if len(keys) == 1 {
					if next.Kind == 8 {
						next.Value = fmt.Sprintf("%v", value)
						return nil
					} else if next.Kind == 4 {
						next.Value = fmt.Sprintf("%v", value)
						next.Kind = 8
						next.Children = nil
						return nil
					} else {
						panic(fmt.Errorf("unexpected condition1 "))
					}
				} else {
					if next.Kind == 4 {
						return NodeSet(next, keys[1:], value)
					} else {
						panic(fmt.Errorf("unexpected condition2"))
					}
				}
			}
		}
	}
	return fmt.Errorf("value not found for path %v", keys)
}

func dumpYaml(node *ynext.Node) {
	d, err := ynext.Marshal(node)
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(os.Stdout,"%s", string(d))
	//for _, c := range node.Children {
	//	dumpYaml(c)
	//}
}

func dumpJson(node *ynext.Node) {
	jsonBytes, err := json.MarshalIndent(node, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(os.Stderr,"dumpJson: %s\n", string(jsonBytes))
}
