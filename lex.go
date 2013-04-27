package toml

import (
	"strings"
	"unicode/utf8"
	"fmt"
	//"unicode"
)

type tokenType int

const (
	tokenError  tokenType = iota
	tokenEOF
	tokenSpace
	tokenKeyGroup
	tokenKey
	tokenKeySep
	tokenBool
	tokenNumber
	tokenString
	tokenDatetime
	tokenArrayStart
	tokenArrayEnd
	tokenArraySep
)

const (
	eof = -1
	keyGroupStart = '['
	keyGroupEnd   = ']'
	keyGroupSep   = '.'
	keySep        = '='
	keySep2        = ':'
	commentStart  = '#'
)

var	datetimeFormat = []rune{
	'0','0','0','0', '-', '0','0', '-', '0','0',
	'T',
	'0', '0', ':', '0', '0', ':', '0', '0',
	'Z',
}

type token struct {
	typ tokenType  // type.
	pos Pos
	val string     // value.
}

func (t token) String() string {
	switch {
	case t.typ == tokenEOF:
		return "EOF"
	case t.typ == tokenError:
		return t.val
	case len(t.val) > 10:
		return fmt.Sprintf("%.10q...", t.val)
	}
	return fmt.Sprintf("%q", t.val)
}

// stateFn represents the state of the scanner as a function that returns the next state.
type stateFn func(*lexer) stateFn

type lexer struct {
	input      string
	state      stateFn
	pos        Pos
	start      Pos
	width      Pos
	lastPos    Pos
	tokens     chan token
	arrayDepth int
}

// lex creates a new scanner for the input string.
func lex(input string) *lexer {
	l := &lexer{
		input:      input,
		tokens:      make(chan token),
	}
	go l.run()
	return l
}

// next returns the next rune in the input.
func (l *lexer) next() rune {
	if int(l.pos) >= len(l.input) {
		l.width = Pos(0)
		return eof
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = Pos(w)
	l.pos += l.width
	return r
}

// peek returns but does not consume the next rune in the input.
func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// backup steps back one rune. Can only be called once per call of next.
func (l *lexer) backup() {
	l.pos -= l.width
}

func (l *lexer) is(word string) bool {
	return strings.HasPrefix(l.input[l.pos-l.width:], word)
}

// accept consumes the next rune if it's from the valid set.
func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}

// acceptRun consumes a run of runes from the valid set.
func (l *lexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.next()) >= 0 {
	}
	l.backup()
}

func (l *lexer) run() {
	for l.state = lexStart; l.state != nil; {
		l.state = l.state(l)
	}
}

// emit passes an token back to the client.
func (l *lexer) emit(t tokenType) {
	l.tokens <- token{t, l.start, l.input[l.start:l.pos]}
	l.start = l.pos
}

// errorf returns an error token and terminates the scan by passing
// back a nil pointer that will be the next state, terminating l.nextItem.
func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.tokens <- token{tokenError, l.start, fmt.Sprintf(format, args...)}
	return nil
}

// ignore skips over the pending input before this point.
func (l *lexer) ignore() {
	l.start = l.pos
}

func (l *lexer) nextToken() token {
	token := <-l.tokens
	l.lastPos = token.pos
	return token
}

func (l *lexer) lineNumber() int {
	return 1 + strings.Count(l.input[:l.lastPos], "\n")
}

func lexStart(l *lexer) stateFn {
	//pd("start", l.peek(), string(l.peek()))
	switch r := l.next(); {
	case r == eof:
		l.emit(tokenEOF)
		return nil
  case isDash(r):
		l.ignore()
		return lexStart
	case isNewLine(r):
    if isNewLine(l.peek()) {
        l.emit(tokenEOF)
        return nil
      }
		l.ignore()
		return lexStart
	case isSpace(r):
		ignoreSpaces(l)
		return lexStart
	case r == commentStart:
		return lexComment(l, lexStart)
	case r == keyGroupStart:
		return lexKeyGroup
	case isAlpha(r):
		return lexKey
	default:
		return l.errorf("lexStart parse error %#U", r)
	}
	return nil
}

func lexComment(l *lexer, nextState stateFn) stateFn {
	for { 
		if r := l.next(); r == '\n' || r == eof {
			l.backup()
			break
		}
	}
	l.ignore()
	return nextState
}

func lexKeyGroup(l *lexer) stateFn { 
Loop:
	for {
		switch r := l.next(); {
		case r == keyGroupEnd:
			break Loop
		case isAlphaNumeric(r) || r == keyGroupSep:
			// absorb.
		default:
			l.backup()
			return l.errorf("bad keygroup name %#U", r)
		}
	}
	l.emit(tokenKeyGroup)
	return lexStart
}

func lexKey(l *lexer) stateFn {
Loop:
	for {
		switch r := l.next(); {
		case isAlphaNumeric(r):
			// absorb.
		case isSpace(r):
			l.backup()
			break Loop
    default:
			l.backup()
			return l.errorf("bad keyname %#U", r)
		}
	}
	l.emit(tokenKey)
	return lexKeySep
}

func lexKeySep(l *lexer) stateFn {
	ignoreSpaces(l)

	r := l.next()
	if r == keySep || r == keySep2 {
		l.emit(tokenKeySep)
		return lexValue
	}
	return l.errorf("bad key seperator %#U, want %#U", r, keySep)
}

func lexValue(l *lexer) stateFn {
	//pd("value %q", l.peek())
	switch r := l.next(); { 
	case r == eof:
		l.emit(tokenEOF)
		return nil
	case r == '\r':
		l.ignore()
		return lexValue
	case r == '\n':
		l.ignore()
		if l.arrayDepth == 0 {
			return lexStart
		} else {
			return lexValue
		}
	case isSpace(r):
		ignoreSpaces(l)
		return lexValue
	case r == commentStart:
		return lexComment(l, lexValue)
	case r == '"':
		return lexString
	case r == '[':
		l.arrayDepth ++
		l.emit(tokenArrayStart)
		return lexValue
	case r == ']':
		l.arrayDepth --
		if l.arrayDepth < 0 {
			return l.errorf("unexpected array end %#U", r)
		}
		l.emit(tokenArrayEnd)
		return lexValue
	case r == ',':
		if l.arrayDepth > 0 {
			l.emit(tokenArraySep)
			return lexValue
		} else {
			return l.errorf("unexpected comma outside array")
		}
	case r == '+' || r == '-':
		l.backup()
		return lexNumber
	case '0' <= r && r <= '9':
		l.backup()
		return lexNumberOrDatetime
	case l.is("true"):
		l.pos += Pos(3)
		l.emit(tokenBool)
		return lexStart
	case l.is("false"):
		l.pos += Pos(4)
		l.emit(tokenBool)
		return lexStart
	default:
		return l.errorf("bad value %#U", r)
	}
	return nil
}

func lexString(l *lexer) stateFn {
Loop:
	for {
		switch l.next() {
		case '\\':
			r := l.next()
			if r == eof {
				return l.errorf("unterminated string")
			}
		case eof, '\n':
			return l.errorf("unterminated string")
		case '"':
			break Loop
		}
	}
	l.emit(tokenString)
	return lexValue
}

func lexNumberOrDatetime(l *lexer) stateFn {
	i := int(l.pos)+4
	if len(l.input) > i && l.input[i] == '-' {
		return lexDatetime
	} 

	return lexNumber
}

func lexNumber(l *lexer) stateFn {
	// Optional leading sign.
	l.accept("+-")
	digits := "0123456789"
	l.acceptRun(digits)
	if l.accept(".") {
		l.acceptRun(digits)
	}
	// Next thing mustn't be alphanumeric or datetime
	if r := l.peek(); isAlphaNumeric(r) || r == '-' {
		l.next()
		return l.errorf("bad number syntax: %q", l.input[l.start:l.pos])
	}

	l.emit(tokenNumber)
	return lexValue
}

func lexDatetime(l *lexer) stateFn {
	for _, f := range datetimeFormat {
		r := l.next()
		if (f == '0' && isDigit(r)) || f == r {
			// absorb.
		} else {
			return l.errorf("bad datetime %#U", r)
		}
	}
	l.emit(tokenDatetime)
	return lexValue
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

func isDash(r rune) bool {
  return r == '-' 
}

func isNewLine(r rune) bool {
	return r == '\n' || r == '\r'
}

func isAlpha(r rune) bool {
	return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isAlphaNumeric(r rune) bool {
	return isAlpha(r) || isDigit(r)
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func ignoreSpaces(l *lexer) {
	for isSpace(l.next()) {
		// absorb.
	}
	l.backup()
	l.ignore()
}
