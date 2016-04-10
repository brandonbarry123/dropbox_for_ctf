// SUPPORT CODE
//
// You shouldn't need to alter
// the contents of this file

package rpc

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"sync"

	"./internal/rpcType"
)

var handlers = make(map[string]handler)
var finalizer func()
var mtx sync.Mutex

var invokeMtx sync.Mutex

// RegisterHandler registers a handler under the given
// name. f should be a function satisfying the following
// requirements:
//
// - f must take 0 or more arguments
//
// - f must not take variadic arguments
//
// - f must return 0 or 1 values
//
// - the argument and return types of f cannot be
// pointers, functions, channels, or interfaces
//
// - if the argument or return types of f contain other
// types (such as structs or arrays), those types
// cannot be pointers, functions, interfaces, or channels,
// nor can they recursively contain pointers, functions,
// interfaces, or channels.
func RegisterHandler(name string, f interface{}) {
	mtx.Lock()
	defer mtx.Unlock()
	if _, ok := handlers[name]; ok {
		panic("handler already registered with given name")
	}
	h, err := getHandler(f)
	if err != nil {
		panic(err)
	}
	handlers[name] = h
}

// RegisterFinalizer registers a function which will
// be called when the server is shut down.
func RegisterFinalizer(f func()) {
	mtx.Lock()
	defer mtx.Unlock()
	if finalizer != nil {
		panic("finalizer already registered")
	}
	finalizer = f
}

// RunServer runs the server. It panics if a finalizer
// has not been registered.
//
// The server listens for incoming method requests,
// handling each one in order. It is guaranteed that
// no two method requests will be handled simultaneously
// (so your handler may assume that no other handlers)
// are executing concurrently with it.
//
// The server runs until it receives SIGINT (ctrl+C
// on the command line), at which point it calls the
// finalizer and returns.
func RunServer(addr string) error {
	mtx.Lock()
	defer mtx.Unlock()
	if finalizer == nil {
		panic(fmt.Errorf("no finalizer registered"))
	}

	l, err := net.Listen("tcp4", addr)
	if err != nil {
		return err
	}

	rpc.Register(&rpcType.Server{request})

	go func() {
		rpc.Accept(l)
	}()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	<-c

	// It would be preferable to shut down the RPC
	// server first, but that is difficult, so instead
	// just take the lock, call the finalizer, and
	// return. No more RPC calls will be able to proceed.
	invokeMtx.Lock()
	finalizer()
	return nil
}

func request(req rpcType.Request, resp *rpcType.Response) error {
	invokeMtx.Lock()
	defer invokeMtx.Unlock()

	h, ok := handlers[req.Name]
	if !ok {
		return fmt.Errorf("no method with name: %v", req.Name)
	}

	return handleRequest(h, req, resp)
}
