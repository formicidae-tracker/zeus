package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jessevdk/go-flags"
)

type Nodename string

var lister *NodeLister

func (n *Nodename) Complete(match string) []flags.Completion {
	nodes, err := lister.ListNodes()
	if err != nil {
		return nil
	}
	res := make([]flags.Completion, 0, len(nodes))
	for name, node := range nodes {
		if strings.HasPrefix(name, match) == false {
			continue
		}
		res = append(res, flags.Completion{
			Item:        name,
			Description: fmt.Sprintf("%s:%d", node.Address, node.Port),
		})
	}
	return res
}

func GetNode(name Nodename) (Node, error) {
	nodes, err := lister.ListNodes()
	if err != nil {
		return Node{}, err
	}
	node, ok := nodes[string(name)]
	if ok == false {
		return Node{}, fmt.Errorf("Could not find node '%s'", name)
	}

	return node, nil

}

func Nodes() ([]Node, error) {
	nodes, err := lister.ListNodes()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(nodes))
	for name, _ := range nodes {
		names = append(names, name)
	}
	sort.Strings(names)
	res := make([]Node, 0, len(nodes))
	for _, n := range names {
		res = append(res, nodes[n])
	}
	return res, nil
}

func init() {
	lister = NewLister()
}
