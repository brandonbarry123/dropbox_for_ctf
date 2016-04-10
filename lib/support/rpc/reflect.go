// SUPPORT CODE
//
// You shouldn't need to alter
// the contents of this file

package rpc

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"reflect"

	"./internal/pool"
	"./internal/rpcType"
)

type handler struct {
	f    reflect.Value
	args []reflect.Type
	ret  *reflect.Type
}

func handleRequest(h handler, req rpcType.Request, resp *rpcType.Response) error {
	if len(req.Args) != len(h.args) {
		return fmt.Errorf("expected %v arguments; got %v", len(h.args), len(req.Args))
	}

	args := make([]reflect.Value, len(h.args))
	for i, arg := range req.Args {
		args[i] = reflect.New(h.args[i]).Elem()
		b := bytes.NewBuffer(arg)
		dec := gob.NewDecoder(b)
		err := dec.DecodeValue(args[i])
		if err != nil {
			return err
		}
	}

	ret := h.f.Call(args)
	if h.ret != nil {
		b := pool.GetBuffer()
		enc := gob.NewEncoder(b)
		err := enc.EncodeValue(ret[0])
		if err != nil {
			return fmt.Errorf("error after calling function: %v", err)
		}
		resp.Return = b.Bytes()
		pool.PutBuffer(b)
	}
	return nil
}

func getHandler(f interface{}) (handler, error) {
	h := handler{f: reflect.ValueOf(f)}
	typ := reflect.TypeOf(f)
	if typ.Kind() != reflect.Func {
		return h, fmt.Errorf("handler has non-function type")
	}

	switch typ.NumOut() {
	case 0:
	case 1:
		h.ret = new(reflect.Type)
		*h.ret = typ.Out(0)
		err := validType(*h.ret)
		if err != nil {
			return h, fmt.Errorf("handler has bad return value: %v", err)
		}
	default:
		return h, fmt.Errorf("handler must have 0 or 1 return values")
	}

	if typ.IsVariadic() {
		return h, fmt.Errorf("handler cannot have a variadic argument")
	}

	h.args = make([]reflect.Type, typ.NumIn())
	for i := 0; i < typ.NumIn(); i++ {
		h.args[i] = typ.In(i)
		err := validType(h.args[i])
		if err != nil {
			return h, err
		}
	}
	return h, nil
}

var validKinds = map[reflect.Kind]bool{
	reflect.Bool:          true,
	reflect.Int:           true,
	reflect.Int8:          true,
	reflect.Int16:         true,
	reflect.Int32:         true,
	reflect.Int64:         true,
	reflect.Uint:          true,
	reflect.Uint8:         true,
	reflect.Uint16:        true,
	reflect.Uint32:        true,
	reflect.Uint64:        true,
	reflect.Uintptr:       true,
	reflect.Float32:       true,
	reflect.Float64:       true,
	reflect.Complex64:     true,
	reflect.Complex128:    true,
	reflect.Array:         true,
	reflect.Map:           true,
	reflect.Slice:         true,
	reflect.String:        true,
	reflect.Struct:        true,
	reflect.UnsafePointer: true,
}

// Disallowed kinds: Chan, Func, Interface, Ptr

func validType(typ reflect.Type) error {
	return validTypeHelper(make(map[reflect.Type]bool), typ)
}

func validTypeHelper(m map[reflect.Type]bool, typ reflect.Type) error {
	if m[typ] {
		return nil
	}
	m[typ] = true
	if !validKinds[typ.Kind()] {
		return fmt.Errorf("cannot handle type: %v", typ)
	}
	switch typ.Kind() {
	case reflect.Array, reflect.Slice:
		err := validTypeHelper(m, typ.Elem())
		if err != nil {
			return fmt.Errorf("bad slice or array element type: %v", err)
		}
	case reflect.Map:
		err := validTypeHelper(m, typ.Key())
		if err != nil {
			return fmt.Errorf("bad map key type: %v", err)
		}
		err = validTypeHelper(m, typ.Elem())
		if err != nil {
			return fmt.Errorf("bad map value type: %v", err)
		}
	case reflect.Struct:
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			err := validTypeHelper(m, field.Type)
			if err != nil {
				return fmt.Errorf("struct field %v: %v", field.Name, err)
			}
		}
	}
	return nil
}
