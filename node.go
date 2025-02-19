package jsonquery

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"

	"github.com/antchfx/xpath"
)

// A NodeType is the type of a Node.
type NodeType uint

const (
	// DocumentNode is a document object that, as the root of the document tree,
	// provides access to the entire XML document.
	DocumentNode NodeType = iota
	// ElementNode is an element.
	ElementNode
	// TextNode is the text content of a node.
	TextNode
)

// A Node consists of a NodeType and some Data (tag name for
// element nodes, content for text) and are part of a tree of Nodes.
type Node struct {
	Parent, PrevSibling, NextSibling, FirstChild, LastChild *Node

	Type NodeType
	Data string

	level int
}

// ChildNodes gets all child nodes of the node.
func (n *Node) ChildNodes() []*Node {
	var a []*Node
	for nn := n.FirstChild; nn != nil; nn = nn.NextSibling {
		a = append(a, nn)
	}
	return a
}

// InnerText gets the value of the node and all its child nodes.
func (n *Node) InnerText() string {
	var output func(*bytes.Buffer, *Node)
	output = func(buf *bytes.Buffer, n *Node) {
		if n.Type == TextNode {
			buf.WriteString(n.Data)
			return
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			output(buf, child)
		}
	}
	var buf bytes.Buffer
	output(&buf, n)
	return buf.String()
}

func outputXML(buf *bytes.Buffer, n *Node) {
	switch n.Type {
	case ElementNode:
		if n.Data == "" {
			buf.WriteString("<element>")
		} else {
			buf.WriteString("<" + n.Data + ">")
		}
	case TextNode:
		buf.WriteString(n.Data)
		return
	}

	for child := n.FirstChild; child != nil; child = child.NextSibling {
		outputXML(buf, child)
	}
	if n.Data == "" {
		buf.WriteString("</element>")
	} else {
		buf.WriteString("</" + n.Data + ">")
	}
}

// OutputXML prints the XML string.
func (n *Node) OutputXML() string {
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0"?>`)
	for n := n.FirstChild; n != nil; n = n.NextSibling {
		outputXML(&buf, n)
	}
	return buf.String()
}

// SelectElement like Query finds the first of child elements 
// matching the specified query. However, it will panic if the
// query cannot be parsed.
func (n *Node) SelectElement(query string) *Node {
	return FindOne(n, query)
}

// SelectElements like QueryAll finds all child elements matching
// the specified query. However, it will panic if the query cannot
// be parsed.
func (n *Node) SelectElements(query string) []*Node {
	return Find(n, query)
}

// Queryfinds the first of child elements 
// matching the specified query.
func (n *Node) Query(query string) (*Node, error) {
	return Query(n, query)
}

// QueryAllfinds all child elements matching
// the specified query.
func (n *Node) QueryAll(query string) ([]*Node, error) {
	return QueryAll(n, query)
}

// QuerySelector returns the first matched child Node by the
// specified XPath selector.
func (n *Node) QuerySelector(selector *xpath.Expr) *Node {
	return QuerySelector(n, selector)
}

// QuerySelectorAll searches all of the Node that matches the
// specified XPath selectors.
func (n *Node) QuerySelectorAll(selector *xpath.Expr) []*Node {
	return QuerySelectorAll(n, selector)
}

// LoadURL loads the JSON document from the specified URL.
func LoadURL(url string) (*Node, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return Parse(resp.Body)
}

func parseValue(x interface{}, top *Node, level int) {
	addNode := func(n *Node) {
		if n.level == top.level {
			top.NextSibling = n
			n.PrevSibling = top
			n.Parent = top.Parent
			if top.Parent != nil {
				top.Parent.LastChild = n
			}
		} else if n.level > top.level {
			n.Parent = top
			if top.FirstChild == nil {
				top.FirstChild = n
				top.LastChild = n
			} else {
				t := top.LastChild
				t.NextSibling = n
				n.PrevSibling = t
				top.LastChild = n
			}
		}
	}
	switch v := x.(type) {
	case []interface{}:
		for _, vv := range v {
			n := &Node{Type: ElementNode, level: level}
			addNode(n)
			parseValue(vv, n, level+1)
		}
	case map[string]interface{}:
		// The Go’s map iteration order is random.
		// (https://blog.golang.org/go-maps-in-action#Iteration-order)
		var keys []string
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			n := &Node{Data: key, Type: ElementNode, level: level}
			addNode(n)
			parseValue(v[key], n, level+1)
		}
	case string:
		n := &Node{Data: v, Type: TextNode, level: level}
		addNode(n)
	case float64:
		s := strconv.FormatFloat(v, 'f', -1, 64)
		n := &Node{Data: s, Type: TextNode, level: level}
		addNode(n)
	case bool:
		s := strconv.FormatBool(v)
		n := &Node{Data: s, Type: TextNode, level: level}
		addNode(n)
	}
}

func parse(b []byte) (*Node, error) {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	doc := &Node{Type: DocumentNode}
	parseValue(v, doc, 1)
	return doc, nil
}

// Parse JSON document.
func Parse(r io.Reader) (*Node, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return parse(b)
}
