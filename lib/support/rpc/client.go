// SUPPORT CODE
//
// You shouldn't need to alter
// the contents of this file

package rpc

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net/rpc"
	"reflect"

	"./internal/pool"
	"./internal/rpcType"
)

// A ServerRemote represents a server which can execute
// methods.
type ServerRemote struct {
	addr string
	c    *rpc.Client
}

// NewServerRemote creates a new ServerRemote for the
// server located at the given network address.
func NewServerRemote(addr string) *ServerRemote {
	return &ServerRemote{addr: addr}
}

func (s *ServerRemote) dial() error {
	if s.c != nil {
		return nil
	}

	var err error
	s.c, err = rpc.Dial("tcp4", s.addr)
	if err != nil {
		s.c = nil
		return err
	}
	return nil
}

// Call calls the named method on the remote server
// with the given arguments. It expects ret to be a
// pointer to a value of the proper return type.
// When the method returns, its return value will
// be placed in the location pointed to by ret.
// If the method has no return value, ret must be
// nil.
func (s *ServerRemote) Call(method string, ret interface{}, args ...interface{}) error {
	if err := s.dial(); err != nil {
		return err
	}

	if ret != nil && reflect.TypeOf(ret).Kind() != reflect.Ptr {
		return fmt.Errorf("local: ret has non-pointer type")
	}

	var req rpcType.Request
	var resp rpcType.Response

	req.Name = method
	req.Args = make([][]byte, len(args))
	b := pool.GetBuffer()
	defer pool.PutBuffer(b)
	enc := gob.NewEncoder(b)

	for i, arg := range args {
		err := enc.Encode(arg)
		if err != nil {
			return fmt.Errorf("local: %v", err)
		}
		req.Args[i] = append([]byte(nil), b.Bytes()...)
		b.Reset()
	}

	err := s.c.Call("Server.Request", req, &resp)
	if err != nil {
		s.c = nil
		return fmt.Errorf("remote: %v", err)
	}

	if ret != nil {
		if len(resp.Return) == 0 {
			return fmt.Errorf("local: expected 1 return value; got 0")
		}
		val := reflect.ValueOf(ret).Elem()
		b = bytes.NewBuffer(resp.Return)
		defer pool.PutBuffer(b)
		dec := gob.NewDecoder(b)
		err = dec.DecodeValue(val)
		if err != nil {
			return fmt.Errorf("local: %v", err)
		}
	} else if len(resp.Return) > 0 {
		return fmt.Errorf("local: expected 0 return values; got 1")
	}

	return nil
}
