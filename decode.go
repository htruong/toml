package toml

import ( 
	"runtime"
	"reflect"
	"strings"
	"time"
	"fmt"
)

var timeType = reflect.TypeOf(time.Time{})

func Unmarshal(data string, v interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
		}
	}()

	tree, e := Parse(data) 
	if e != nil { return e }

	d := &decode{}

	rv := reflect.Indirect(reflect.ValueOf(v))
	if rv.Kind() == reflect.Interface && rv.NumMethod() == 0 {
		// Decoding into nil interface.
		newv := reflect.ValueOf(make(map[string]interface{}))
		d.top(newv, tree.Root)
		rv.Set(newv)
	} else {
		d.top(reflect.Indirect(reflect.ValueOf(v)), tree.Root)
	}

	return 
}

// An UnmarshalTypeError describes a JSON value that was
// not appropriate for a value of a specific Go type.
type UnmarshalTypeError struct {
	Value string       // description of JSON value - "bool", "array", "number -5"
	Type  reflect.Type // type of Go value it could not be assigned to
}

func (e *UnmarshalTypeError) Error() string {
	return "toml: cannot unmarshal " + e.Value + " into Go value of type " + e.Type.String()
}

type decode struct {
	node Node           // current node
}

// error aborts the decoding by panicking with err.
func (d *decode) error(arg interface{}) {
	panic(arg)
}

// error aborts the decoding by panicking with err.
func (d *decode) errorf(format string, args ...interface{}) {
	panic(fmt.Errorf(format, args...))
}

func (d *decode) top(v reflect.Value, node *ListNode) {
	for _, node := range node.Nodes {
		switch node := node.(type) {
		case *EntryGroupNode:
			for _, key := range node.KeyGroup.StringKeys() {
				var ok bool
				v, ok = d.findField("keygroup", v, key)
				if !ok {
					return
				}
			}
			for _, node := range node.Entries.Nodes {
				d.entry(v, node.(*EntryNode))
			}
		case *EntryNode:
			d.entry(v, node)
		}
	}
}

func (d *decode) findField(context string, v reflect.Value, key string) (next reflect.Value, ok bool) {
	// Check type of target: struct or map[string]T
	switch v.Kind() {
	case reflect.Map:
		t := v.Type()
		if t.Key().Kind() != reflect.String {
			d.error(&UnmarshalTypeError{context, v.Type()})
		}
		// init map
		if v.IsNil() {
			v.Set(reflect.MakeMap(v.Type()))
		}
	case reflect.Struct:
		// continue.
	default:
		d.error(&UnmarshalTypeError{context, v.Type()})
	}

	// Map. for entry only.
	if v.Kind() == reflect.Map {
		if context == "keygroup" {
			next = reflect.ValueOf(make(map[string]interface{}))
			v.SetMapIndex(reflect.ValueOf(key), next)
			return next, true
		} else { // entry
			next = reflect.New(v.Type().Elem()).Elem()
			return next, true
		}
	}

	// Struct
	t := v.Type()
	for i := 0; i < t.NumField(); i ++ {
		tf := t.Field(i)
		name := tf.Tag.Get("toml")
		if name == "" {
			name = tf.Name
		}
		if name == key || strings.EqualFold(name, key) {
			f := v.Field(i)
			if !f.CanSet() {
				continue
			}
			return f, true
		}
	}
	// can't find the field
	return reflect.ValueOf(nil), false
}

func (d *decode) entry(v reflect.Value, node *EntryNode) {
	key := node.Key.Key
	f, ok := d.findField("entry", v, key)
	if !ok {
		return
	}
	d.value(f, node.Value)
	// Write to map, if using struct, f points into struct already.
	if v.Kind() == reflect.Map {
		v.SetMapIndex(reflect.ValueOf(key), f)
	}
}

func (d *decode) value(v reflect.Value, node Node) {
	switch n := node.(type) {
	case *BoolNode:
		value := n.True
		switch v.Kind() {
		case reflect.Bool:
			v.SetBool(value)
		case reflect.Interface:
			if v.NumMethod() == 0 {
				v.Set(reflect.ValueOf(value))
			} else {
				d.error(&UnmarshalTypeError{"bool", v.Type()})
			}
		default:
			d.error(&UnmarshalTypeError{"bool", v.Type()})
		}
	case *StringNode:
		value := n.Text
		switch v.Kind() {
		case reflect.String:
			v.SetString(value)
		case reflect.Interface:
			if v.NumMethod() == 0 {
				v.Set(reflect.ValueOf(value))
			} else {
				d.error(&UnmarshalTypeError{"string", v.Type()})
			}
		default:
			d.error(&UnmarshalTypeError{"string", v.Type()})
		}
	case *NumberNode:
		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if !n.IsInt {
				d.error(&UnmarshalTypeError{"int", v.Type()})
			}
			v.SetInt(n.Int)
		case reflect.Float32, reflect.Float64:
			if !n.IsFloat {
				d.error(&UnmarshalTypeError{"float", v.Type()})
			}
			v.SetFloat(n.Float)
		case reflect.Interface:
			if v.NumMethod() == 0 {
				if n.IsInt {
					v.Set(reflect.ValueOf(n.Int))
					//pd("int %s %p", v, v)
				} else {
					v.Set(reflect.ValueOf(n.Float))
				}
			} else {
				d.error(&UnmarshalTypeError{"number", v.Type()})
			}
		default:
			d.error(&UnmarshalTypeError{"number", v.Type()})
		}
	case *DatetimeNode:
		value := reflect.ValueOf(n.Time)
		switch k := v.Kind(); {
		case k == reflect.Struct && v.Type() == timeType:
			v.Set(value)
		case k == reflect.Interface:
			if v.NumMethod() == 0 {
				v.Set(value)
			} else {
				d.error(&UnmarshalTypeError{"datetime", v.Type()})
			}
		default:
			d.error(&UnmarshalTypeError{"datetime", v.Type()})
		}
	case *ArrayNode:
		switch v.Kind() {
		case reflect.Interface:
			l := len(n.Array.Nodes)
			if v.NumMethod() == 0 {
				newv := reflect.ValueOf(make([]interface{}, l))
				d.value(newv, n)
				v.Set(newv)
			} else {
				d.error(&UnmarshalTypeError{"array", v.Type()})
			}
		case reflect.Array, reflect.Slice:
			l := len(n.Array.Nodes)
			if v.Len() < l {
				if v.Kind() == reflect.Array { 
					d.errorf("run out of fixed array, len is %d, want %d -- %s", v.Len(), l, n)
				}
				// Growing slice
				newv := reflect.MakeSlice(v.Type(), l, l)
				reflect.Copy(newv, v)
				v.Set(newv)
				v.SetLen(l)
			}
			for i, subn := range n.Array.Nodes {
				d.value(v.Index(i), subn)
			}
		default:
			d.error(&UnmarshalTypeError{"array", v.Type()})
		}
	}
}
