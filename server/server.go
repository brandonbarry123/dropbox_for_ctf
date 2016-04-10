package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"../internal"
	"../lib/support/rpc"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %v <listen-address>\n", os.Args[0])
		os.Exit(1)
	}

	// EXAMPLE CODE
	//
	// This code is meant as an example of how to use
	// our framework, not as stencil code. It is not
	// meant as a suggestion of how you should write
	// your application.

	listenAddr := os.Args[1]

	rpc.RegisterHandler("add", addHandler)
	rpc.RegisterHandler("mult", multHandler)
	rpc.RegisterHandler("noOp", noOpHandler)

	rpc.RegisterHandler("upload", uploadHandler)
	rpc.RegisterHandler("download", downloadHandler)
	rpc.RegisterHandler("list", listHandler)
	rpc.RegisterHandler("mkdir", mkdirHandler)
	rpc.RegisterHandler("remove", removeHandler)
	rpc.RegisterHandler("pwd", pwdHandler)
	rpc.RegisterHandler("cd", cdHandler)
	rpc.RegisterFinalizer(finalizer)
	err := rpc.RunServer(listenAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not run server: %v\n", err)
		os.Exit(1)
	}
}

func addHandler(a, b int) int  { return a + b }
func multHandler(a, b int) int { return a * b }
func noOpHandler()             {}

// An implementation of a basic server. This implementation
// is absurdly insecure, and is only meant as an example of
// how to implement the methods required by the example client
// provided in client/client.go; it should not be taken as
// a suggestion of how to design your server.

func uploadHandler(path string, body []byte) string {
	err := ioutil.WriteFile(path, body, 0664)
	if err != nil {
		return err.Error()
	}
	return ""
}

func downloadHandler(path string) internal.DownloadReturn {
	body, err := ioutil.ReadFile(path)
	if err != nil {
		return internal.DownloadReturn{Err: err.Error()}
	}
	return internal.DownloadReturn{Body: body}
}

func listHandler(path string) internal.ListReturn {
	fis, err := ioutil.ReadDir(path)
	if err != nil {
		return internal.ListReturn{Err: err.Error()}
	}
	var entries []internal.DirEnt
	for _, fi := range fis {
		entries = append(entries, internal.DirEnt{
			IsDir_: fi.IsDir(),
			Name_:  fi.Name(),
		})
	}
	return internal.ListReturn{Entries: entries}
}

func mkdirHandler(path string) string {
	err := os.Mkdir(path, 0775)
	if err != nil {
		return err.Error()
	}
	return ""
}

func removeHandler(path string) string {
	err := os.Remove(path)
	if err != nil {
		return err.Error()
	}
	return ""
}

func pwdHandler() internal.PWDReturn {
	path, err := os.Getwd()
	if err != nil {
		return internal.PWDReturn{Err: err.Error()}
	}
	return internal.PWDReturn{Path: path}
}

func cdHandler(path string) string {
	err := os.Chdir(path)
	if err != nil {
		return err.Error()
	}
	return ""
}

func finalizer() {
	fmt.Println("Shutting down...")
}
