package common

import (
	"math/rand"
)

type Node struct {
	Name      string
	Kind      string
	Num       int64
	Childrens map[string]*Node
	Keys      []string
	Data      map[string]int64
	parent    *Node
}

func (n *Node) AddKey(key string, limit int) {
	if length := len(n.Keys); length >= limit {
		n.Keys[rand.Intn(length)] = key
	} else {
		n.Keys = append(n.Keys, key)
	}
}

func (n *Node) AddKeys(keys []string, limit int) {
	for _, key := range keys {
		n.AddKey(key, limit)
	}
}

func newNode(name, kind string, parent *Node) *Node {
	return &Node{
		Name:      name,
		Kind:      kind,
		parent:    parent,
		Childrens: make(map[string]*Node, 256),
		Keys:      make([]string, 0, 64),
		Data:      map[string]int64{},
	}
}
