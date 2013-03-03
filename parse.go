package toml

import (
	"time"
	"strconv"
	"runtime"
	"strings"
	"fmt"
)

type Tree struct {
	Root      *ListNode // top-level root of the tree.
	text      string
	lex       *lexer
	token     [3]token   // three-token lookahead for parser.
	peekCount int
}

func Parse(text string) (tree *Tree, err error) {
	defer parseRecover(&err)

	t := &Tree{}
	t.text = text
	t.lex = lex(text)
	t.parse()

	return t, nil
}

// recover is the handler that turns panics into returns from the top level of Parse.
func parseRecover(errp *error) {
	e := recover()
	if e != nil {
		if _, ok := e.(runtime.Error); ok {
			panic(e)
		}
		*errp = e.(error)
	}
	return
}

// next returns the next tok.
func (t *Tree) next() token {
	if t.peekCount > 0 {
		t.peekCount--
	} else {
		t.token[0] = t.lex.nextToken()
	}
	return t.token[t.peekCount]
}

// backup backs the input stream up one tok.
func (t *Tree) backup() {
	t.peekCount++
}

// backup2 backs the input stream up two tokens.
// The zeroth token is already there.
func (t *Tree) backup2(t1 token) {
	t.token[1] = t1
	t.peekCount = 2
}

// backup3 backs the input stream up three tokens
// The zeroth token is already there.
func (t *Tree) backup3(t2, t1 token) { // Reverse order: we're pushing back.
	t.token[1] = t1
	t.token[2] = t2
	t.peekCount = 3
}

// peek returns but does not consume the next tok.
func (t *Tree) peek() token {
	if t.peekCount > 0 {
		return t.token[t.peekCount-1]
	}
	t.peekCount = 1
	t.token[0] = t.lex.nextToken()
	return t.token[0]
}

// nextNonSpace returns the next non-space tok.
func (t *Tree) nextNonSpace() (tok token) {
	for {
		tok = t.next()
		if tok.typ != tokenSpace {
			break
		}
	}
	//pd("next %d %s", tok.typ, tok.val)
	return tok
}

// peekNonSpace returns but does not consume the next non-space tok.
func (t *Tree) peekNonSpace() (tok token) {
	for {
		tok = t.next()
		if tok.typ != tokenSpace {
			break
		}
	}
	t.backup()
	return tok
}

// Parsing.

// ErrorContext returns a textual representation of the location of the node in the input text.
func (t *Tree) ErrorContext(n Node) (location, context string) {
	pos := int(n.Position())
	text := t.text[:pos]
	byteNum := strings.LastIndex(text, "\n")
	if byteNum == -1 {
		byteNum = pos // On first line.
	} else {
		byteNum++ // After the newline.
		byteNum = pos - byteNum
	}
	lineNum := 1 + strings.Count(text, "\n")
	// TODO
	//context = n.String()
	context = "TODO"
	if len(context) > 20 {
		context = fmt.Sprintf("%.20s...", context)
	}
	return fmt.Sprintf("%d:%d", lineNum, byteNum), context
}

// errorf formats the error and terminates processing.
func (t *Tree) errorf(format string, args ...interface{}) {
	t.Root = nil
	format = fmt.Sprintf("%d: syntax error: %s", t.lex.lineNumber(), format)
	panic(fmt.Errorf(format, args...))
}

// error terminates processing.
func (t *Tree) error(err error) {
	t.errorf("%s", err)
}

// expect consumes the next token and guarantees it has the required type.
func (t *Tree) expect(expected tokenType, context string) token {
	tok := t.nextNonSpace()
	if tok.typ != expected {
		t.unexpected(tok, context)
	}
	return tok
}

// expectOneOf consumes the next token and guarantees it has one of the required types.
func (t *Tree) expectOneOf(expected1, expected2 tokenType, context string) token {
	tok := t.nextNonSpace()
	if tok.typ != expected1 && tok.typ != expected2 {
		t.unexpected(tok, context)
	}
	return tok
}

// unexpected complains about the token and terminates processing.
func (t *Tree) unexpected(tok token, context string) {
	t.errorf("unexpected %s in %s", tok, context)
}

func (t *Tree) parse() Node {
	t.Root = newList(t.peek().pos)

	for t.peek().typ != tokenEOF {
		n := t.top()
		t.Root.append(n)
	}

	return nil
}

// key = value
// [keygroup]
func (t *Tree) top() Node {
	switch tok := t.peekNonSpace(); tok.typ {
	case tokenError:
		t.nextNonSpace()
		t.errorf("%s", tok.val)
	case tokenKeyGroup:
		return t.entryGroup()
	case tokenKey:
		return t.entry()
	default:
		t.errorf("unexpected %q", tok.val)
		return nil
	}
	return nil
}

// [keygroup]
//   ...
func (t *Tree) entryGroup() Node {
	token := t.nextNonSpace()
	keyGroup := parseKeyGroup(token)
	entries := newList(t.peek().pos)

Loop:
	for {
		switch tok := t.peekNonSpace(); tok.typ {
		case tokenKey:
			entries.append(t.entry())
		default:
			break Loop
		}
	}

	return newEntryGroup(token.pos, keyGroup, entries) 
}

// "[foo.bar]"
func parseKeyGroup(tok token) *KeyGroupNode {
	text := tok.val
	name := text[1:len(text)-1]
	keys := newList(tok.pos+Pos(1))

	for _, v := range strings.Split(name, ".") {
		keys.append(newKey(tok.pos+Pos(len(v)), v))
	}

	return newKeyGroup(tok.pos, keys, text)
}

// key = value
func (t *Tree) entry() Node {
	tok := t.nextNonSpace()
	key := newKey(tok.pos, tok.val)
	//pd("entry %s", tok.val)
	t.expect(tokenKeySep, "key seperator")

	return newEntry(tok.pos, key, t.value())
}

// value: string, array, ... 
func (t *Tree) value() Node {
	switch tok := t.nextNonSpace(); tok.typ {
	case tokenBool:
		return newBool(tok.pos, tok.val == "true")
	case tokenNumber:
		v, err := newNumber(tok.pos, tok.val)
		if err != nil { t.error(err) }
		return v
	case tokenString:
		//pd("str %d %s", tok.typ, tok.val)
		v, err := strconv.Unquote(tok.val)
		if err != nil { t.error(err) }
		return newString(tok.pos, v, tok.val)
	case tokenDatetime:
		v, err := time.Parse(time.RFC3339, tok.val)
		if err != nil { t.error(err) }
		return newDatetime(tok.pos, v)
	case tokenArrayStart:
		return t.array() 
	default:
		t.errorf("unexpected %q in value", tok.val)
		return nil
	}
	return nil
}

// [1, 2]
func (t *Tree) array() Node {
	pos := t.peek().pos
	array := newList(pos)
Loop:
	for {
		switch tok := t.peekNonSpace(); tok.typ {
		case tokenArrayEnd:
			t.nextNonSpace()
			break Loop
		default:
			//pd("array %s", tok.val)
			node := t.value()
			if t.peekNonSpace().typ != tokenArrayEnd {
				t.expect(tokenArraySep, "array")
			}
			array.append(node)
		}
	}

	return newArray(pos, array)
}
