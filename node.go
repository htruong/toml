package toml

import (
	"time"
	"strconv"
	"fmt"
	"strings"
	"bytes"
)

type Node interface {
	Type() NodeType
	String() string
	//Copy() Node
	Position() Pos // byte position of start of node in full original input string
	unexported()
}

type NodeType int

type Pos int

const (
	NodeList          NodeType = iota
	NodeEntryGroup
	NodeKeyGroup
	NodeEntry
	NodeKey                           // Key name
	NodeBool                          // A boolean constant.
	NodeString                        // A string constant.
	NodeNumber                        // A number constant.
	NodeDatetime                      // A datetime constant.
	NodeArray                         
)

func (t NodeType) Type() NodeType {
	return t
}

func (p Pos) Position() Pos {
	return p
}

func (p *Pos) unexported() {
}

// Nodes.

type ListNode struct {
	NodeType
	Pos
	Nodes []Node // The element nodes in lexical order.
}

func newList(pos Pos) *ListNode {
	return &ListNode{NodeType: NodeList, Pos: pos}
}

func (l *ListNode) append(n Node) {
	l.Nodes = append(l.Nodes, n)
}

func (l ListNode) String() string {
	b := new(bytes.Buffer)
	for _, n := range l.Nodes {
		fmt.Fprint(b, n)
	}
	return b.String()
}

type EntryGroupNode struct {
	NodeType
	Pos
	KeyGroup   *KeyGroupNode
	Entries    *ListNode
}

func newEntryGroup(pos Pos, keyGroup *KeyGroupNode, entries *ListNode) *EntryGroupNode {
	return &EntryGroupNode{NodeType: NodeEntryGroup, Pos: pos, KeyGroup: keyGroup, Entries: entries}
}

func (g EntryGroupNode) String() string {
	entries := []string{}
	for _, e := range g.Entries.Nodes {
		entries = append(entries, e.String())
	}
	return fmt.Sprintf("%s\n%s", g.KeyGroup, strings.Join(entries, "\n"))
}

type KeyGroupNode struct {
	NodeType
	Pos
	Keys   *ListNode
	Text   string
}

func newKeyGroup(pos Pos, keys *ListNode, text string) *KeyGroupNode {
	return &KeyGroupNode{NodeType: NodeKeyGroup, Pos: pos, Keys: keys, Text: text}
}

func (g KeyGroupNode) String() string {
	keys := []string{}
	for _, k := range g.Keys.Nodes {
		keys = append(keys, k.String())
	}
	return fmt.Sprintf("[%s]", strings.Join(keys, "."))
}

func (g *KeyGroupNode) StringKeys() []string {
	keys := []string{}
	for _, n := range g.Keys.Nodes {
		keys = append(keys, n.(*KeyNode).Key)
	}
	return keys
}

type EntryNode struct {
	NodeType
	Pos
	Key     *KeyNode
	Value   Node
}

func newEntry(pos Pos, key *KeyNode, value Node) *EntryNode {
	return &EntryNode{NodeType: NodeEntry, Pos: pos, Key: key, Value: value}
}

func (e EntryNode) String() string {
	return fmt.Sprintf("%s = %s", e.Key, e.Value)
}

type KeyNode struct {
	NodeType
	Pos
	Key    string
}

func newKey(pos Pos, key string) *KeyNode {
	return &KeyNode{NodeType: NodeKey, Pos: pos, Key: key}
}

func (k KeyNode) String() string {
	return k.Key
}

type BoolNode struct {
	NodeType
	Pos
	True bool
}

func newBool(pos Pos, true bool) *BoolNode {
	return &BoolNode{NodeType: NodeBool, Pos: pos, True: true}
}

func (b BoolNode) String() string {
	if b.True {
		return "true"
	}
	return "false"
}

type StringNode struct {
	NodeType
	Pos
	Text  string       // The string, after quote processing.
	Quoted string      // The original text of the string, with quotes. 
}

func newString(pos Pos, text, orig string) *StringNode {
	return &StringNode{NodeType: NodeString, Pos: pos, Text: text, Quoted: orig}
}

func (s StringNode) String() string {
	return s.Quoted
}

type NumberNode struct {
	NodeType
	Pos
	IsInt      bool       // Number has an integral value.
	IsFloat    bool       // Number has a floating-point value.
	Int        int64      // The signed integer value.
	Float      float64    // The floating-point value.
	Text       string     // The original textual representation from the input.
}

func newNumber(pos Pos, text string) (*NumberNode, error) {
	n := &NumberNode{NodeType: NodeNumber, Pos: pos, Text: text}
	i, err := strconv.ParseInt(text, 0, 64)
	if err == nil {
		n.IsInt = true
		n.Int = i
		return n, nil
	} 

	f, err := strconv.ParseFloat(text, 64)
	if err == nil {
		n.IsFloat = true
		n.Float = f
		return n, nil
	}

	return nil, fmt.Errorf("illegal number syntax: %q", text)
}

func (n NumberNode) String() string {
	return n.Text
}

type DatetimeNode struct {
	NodeType
	Pos
	Time time.Time
}

func newDatetime(pos Pos, time time.Time) *DatetimeNode { 
	return &DatetimeNode{NodeType: NodeDatetime, Pos: pos, Time: time}
}

func (t DatetimeNode) String() string {
	return t.Time.Format(time.RFC3339)
}

type ArrayNode struct { 
	NodeType
	Pos
	Array *ListNode
}

func newArray(pos Pos, array *ListNode) *ArrayNode {
	return &ArrayNode{NodeType: NodeArray, Pos: pos, Array: array} 
}

func (a ArrayNode) String() string {
	values := []string{}
	for _, v := range a.Array.Nodes {
		values = append(values, v.String())
	}
	return fmt.Sprintf("[%s]", strings.Join(values, ", "))
}
