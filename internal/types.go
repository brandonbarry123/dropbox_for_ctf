package internal

// EXAMPLE CODE
//
// This code is meant as an example of how to use
// our framework, not as stencil code. It is not
// meant as a suggestion of how you should write
// your application.

// This type is returned by a method on the server,
// so it has to be accessible from both the server
// (so it can return it) and the client (so it can
// use the type once it gets the method's return
// value). Thus, put it here in this shared library.
type DirEnt struct {
	IsDir_ bool   // True if the entry is a directory; false if it is a file
	Name_  string // Name of the entry
}

// DirEnt implements the client.DirEnt interface.
func (d DirEnt) IsDir() bool  { return d.IsDir_ }
func (d DirEnt) Name() string { return d.Name_ }

// This type is returned by a method on the server,
// so it has to be accessible from both the server
// (so it can return it) and the client (so it can
// use the type once it gets the method's return
// value). Thus, put it here in this shared library.
type ListReturn struct {
	Entries []DirEnt
	Err     string // If no error was encountered, this will be empty
}

// This type is returned by a method on the server,
// so it has to be accessible from both the server
// (so it can return it) and the client (so it can
// use the type once it gets the method's return
// value). Thus, put it here in this shared library.
type PWDReturn struct {
	Path string
	Err  string // If no error was encountered, this will be empty
}

// This type is returned by a method on the server,
// so it has to be accessible from both the server
// (so it can return it) and the client (so it can
// use the type once it gets the method's return
// value). Thus, put it here in this shared library.
type DownloadReturn struct {
	Body []byte
	Err  string // If no error was encountered, this will be empty
}

type AuthReturn struct {
        Auth bool
        Session string
}







