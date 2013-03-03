package toml

import (
	"testing"
)

var doc3 = `
A = 1
[User]
Name = "guten"
`

type User struct {
	Name string
}

type Rc struct {
	A int
	User User
}

func TestDecode(t *testing.T) {
	var rc Rc
	//var rc interface{}
	//var rc map[string]interface{}
	//rc = Rc{}
	//rc = interface{}{}
	//rc = make(map[string]interface{})
	e := Unmarshal(doc3, &rc)
	if e != nil { panic(e) }
	pd(rc)
}
