package toml

import (
	"testing"
)

var doc = `
# This is a TOML document. Boom.

[owner] 
[owner] # Whoa there.
andrew = "gallant # poopy" # weeeee
predicate = false
num = -5192
f = -0.5192
zulu = 1979-05-27T07:32:00Z
whoop = "poop"
arrs = [
	1987-07-05T05:45:00Z,
	5,
	"wat?",
	"hehe \n\r kewl",
	[6], [],
	5.0,
	# sweetness
] # more comments
# hehe
`


func TestLex(t *testing.T) {
	l := lex(doc)
	for {
		c := <- l.tokens
		if c.typ == tokenEOF {
			//pd(c)
			break
		}
		//pd(c)
	}
}
