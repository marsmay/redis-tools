package common

import (
	"sort"
	"strings"

	"github.com/marsmay/golib/strings2"
)

type Tree struct {
	Nodes     map[string]*Node
	separator string
	keysLen   int
	mergeLen  int
	dataSeter func(*Node, map[string]int64)
}

func (t *Tree) merge(node *Node) {
	if len(node.Childrens) == 0 {
		return
	}

	for _, n := range node.Childrens {
		t.merge(n)

		node.Num += n.Num
		node.AddKeys(n.Keys, t.keysLen)

		for k, v := range n.Data {
			node.Data[k] += v
		}
	}

	node.Childrens = nil
}

func (t *Tree) AddNode(key, kind string, data map[string]int64) {
	items := strings.Split(key, t.separator)

	sort.SliceStable(items, func(i, j int) bool {
		iIsNum, jIsNum := strings2.IsNum(items[i]), strings2.IsNum(items[j])

		if iIsNum == jIsNum {
			return i < j
		}

		return jIsNum
	})

	prefix := kind + ":" + items[0]

	if t.Nodes[prefix] == nil {
		t.Nodes[prefix] = newNode(items[0], kind, nil)
	}

	currNode := t.Nodes[prefix]

	for _, name := range items[1:] {
		if currNode.Childrens == nil {
			break
		}

		if currNode.Childrens[name] == nil {
			currNode.Childrens[name] = newNode(name, kind, currNode)
		}

		currNode = currNode.Childrens[name]

		if currNode.parent != nil && len(currNode.parent.Childrens) >= t.mergeLen {
			currNode = currNode.parent
			t.merge(currNode)
		}
	}

	currNode.Num++
	currNode.AddKey(key, t.keysLen)

	if t.dataSeter != nil && data != nil {
		t.dataSeter(currNode, data)
	}
}

func NewTree(separator string, keysLen, mergeLen int, dataSeter func(*Node, map[string]int64)) *Tree {
	return &Tree{
		Nodes:     make(map[string]*Node, 256),
		separator: separator,
		keysLen:   keysLen,
		mergeLen:  mergeLen,
		dataSeter: dataSeter,
	}
}
