// copy from "github.com/GutenYe/tagen.go/pd"
package toml

// Usage
//    
//   Pd(1, 2, 3)         -> "1 2 3"
//   Pd("%d %d", 1, 2)   -> "1 2"

import (
	"fmt"
	"reflect"
	"strings"
)

func pd(values ...interface{}) (n int, err error) {
	if len(values) <= 0 { return }

	v := reflect.ValueOf(values[0])
	if v.Kind() == reflect.String && strings.Contains(v.String(), "%") {
		n, err = fmt.Printf(v.String()+"\n", values[1:]...)
	} else {
		n, err = fmt.Println(values...)
	}

	return n, err
}
