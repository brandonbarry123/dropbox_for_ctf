package main

import (
	"fmt"
	"os"
	"bufio"
	"strings"
	"../internal"
	"../lib/support/client"
	"../lib/support/rpc"
)

var sessionid string
var user string
var currdir string

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %v <server>\n", os.Args[0])
		os.Exit(1)
	}

	server := rpc.NewServerRemote(os.Args[1])
	c := Client{server}
	fmt.Print("Welcome to CS166 Dropbox!")
	redisplay := displayoptions(server)
	for redisplay == true {
		redisplay = displayoptions(server)
	}
	err := client.RunCLI(&c)
	if err != nil {
		// don't actually log the error; it's already been
		// printed by client.RunCLI
		os.Exit(1)
	}
}



func displayoptions(server *rpc.ServerRemote) bool {
	reader := bufio.NewReader(os.Stdin)
        fmt.Print("Please select an option...\n1...Log in to an existing account\n2...Create a new account\n")
        chosenoption, readErr := reader.ReadString('\n')
        if readErr != nil {
                fmt.Fprintf(os.Stderr, "error reading option: %v\n", readErr)
                os.Exit(1)
        }
	
	
	switch chosenoption {
	case "1\n":
		var found_creds bool
        	found_creds = AskCreds(server)
        	for found_creds != true {
                	fmt.Fprintf(os.Stderr, "Wrong credentials!\n")
                	found_creds = AskCreds(server)
        	}
	case "2\n":
		//sign up
		signedUp := newUserDetails(server)
        	for signedUp == false {
        	        signedUp = newUserDetails(server)
	        }
		fmt.Print("New user created!\n")
		return true
	default:
		return true
	}

	return false
}



func newUserDetails(server *rpc.ServerRemote) bool {
	reader := bufio.NewReader(os.Stdin)
        fmt.Print("Enter new username: ")
        username, readErr := reader.ReadString('\n')
        if readErr != nil {
                fmt.Fprintf(os.Stderr, "error reading username: %v\n", readErr)
                return false
        }
	
	
        fmt.Print("Enter new Password: ")
        password, readErr := reader.ReadString('\n')
        if readErr != nil {
                fmt.Fprintf(os.Stderr, "error reading password: %v\n", readErr)
                return false
        }
	fmt.Print("Confirm new password: ")
        password_confirm, readErr := reader.ReadString('\n')
        if readErr != nil {
                fmt.Fprintf(os.Stderr, "error reading password: %v\n", readErr)
                return false
        } else {
                for password != password_confirm {
                        fmt.Fprintf(os.Stderr, "Password does not match!")
                        fmt.Print("Confirm new password: ")
        		password_confirm, readErr = reader.ReadString('\n')
			if readErr != nil {
         		       fmt.Fprintf(os.Stderr, "error reading password: %v\n", readErr)
               		       return false
       			 }
                }
        }
	
        var signup bool
        err := server.Call("signup", &signup, strings.TrimRight(username, " \r\n"), strings.TrimRight(password, " \r\n"))
        if err != nil {
                fmt.Fprintf(os.Stderr, "error authenticating: %v\n", err)
                return false
        }
	if signup == false {
		fmt.Print("Username already exists, or your username is not between 5 and 16 characters, or your password is less than 6 characters!\n")
	}
        return signup	
}




func AskCreds(server *rpc.ServerRemote) bool {
	reader := bufio.NewReader(os.Stdin)
        fmt.Print("Enter username: ")
        username, readErr := reader.ReadString('\n')
	if readErr != nil {
                fmt.Fprintf(os.Stderr, "error reading username: %v\n", readErr)
                return false
        }
        fmt.Print("Enter Password: ")
        password, readErr := reader.ReadString('\n')
	if readErr != nil {
		fmt.Fprintf(os.Stderr, "error reading password: %v\n", readErr)
                return false
	}
	user = strings.TrimRight(username, " \r\n")	
	currdir = "./userfs/" + user + "/"

        var ret internal.AuthReturn
        err := server.Call("authenticate", &ret, strings.TrimRight(username, " \r\n"), strings.TrimRight(password, " \r\n"))
	if err != nil {
                fmt.Fprintf(os.Stderr, "error authenticating: %v\n", err)
                return false
        }
	sessionid = ret.Session
	return ret.Auth
	
}  

// An implementation of a basic client to match the example server
// implementation. This client/server implementation is absurdly
// insecure, and is only meant as an example of how to implement
// the client.Client interface; it should not be taken as a suggestion
// of how to design your client or server.
type Client struct {
	server *rpc.ServerRemote
}


func (c *Client) Chperm(path string, sharee string, perm string) (err error) {
	var ret string
   	     
        err = c.server.Call("chperm", &ret, currdir + path, sharee, perm, user, sessionid)
        if err != nil {
                return client.MakeFatalError(err)
        }
        if ret != "" { 
                if(ret == "reauth"){
                        fmt.Print("Your session has expired. Please log in again.\n")
                        os.Exit(1)
                }
                return fmt.Errorf(ret)
        }
        
        return nil
}





func (c *Client) Share(path string, sharee string, perm string) (err error) {
	var ret string
        err = c.server.Call("share", &ret, currdir + path, sharee, perm, user, sessionid)
        if err != nil {
                return client.MakeFatalError(err)
        }
        if ret != "" {
                if(ret == "reauth"){
                        fmt.Print("Your session has expired. Please log in again.\n")
                        os.Exit(1)
                }
                return fmt.Errorf(ret)
        }
        return nil
}


func (c *Client) Unshare(path string, sharee string) (err error) {
	var ret string

        err = c.server.Call("unshare", &ret, currdir + path, sharee, user, sessionid)
        if err != nil {
                return client.MakeFatalError(err)
        }
        if ret != "" {
                if(ret == "reauth"){
                        fmt.Print("Your session has expired. Please log in again.\n")
                        os.Exit(1)
                }
                return fmt.Errorf(ret)
        }

        return nil
}












func (c *Client) Upload(path string, body []byte) (err error) {
	var ret string
	err = c.server.Call("upload", &ret, currdir + path, user, body, sessionid)
	if err != nil {
		return client.MakeFatalError(err)
	}
	if ret != "" {
		if(ret == "reauth"){
                        fmt.Print("Your session has expired. Please log in again.\n")
                        os.Exit(1)
                }
		return fmt.Errorf(ret)
	}
	return nil
}

func (c *Client) Download(path string) (body []byte, err error) {
	var ret internal.DownloadReturn
	err = c.server.Call("download", &ret, currdir+path, user, sessionid)
	if err != nil {
		return nil, client.MakeFatalError(err)
	}
	if ret.Err != "" {
		if(ret.Err == "reauth"){
                        fmt.Print("Your session has expired. Please log in again.\n")
                        os.Exit(1)
                }
		return nil, fmt.Errorf(ret.Err)
	}
	return ret.Body, nil
}

func (c *Client) List(path string) (entries []client.DirEnt, err error) {
	var ret internal.ListReturn
	if path == "" {
                err = c.server.Call("list", &ret, currdir, user, sessionid)
        } else {
                err = c.server.Call("list", &ret, currdir + path, user, sessionid)
        }
	if err != nil {
		return nil, client.MakeFatalError(err)
	}
	if ret.Err != "" {
		if(ret.Err == "reauth"){
                        fmt.Print("Your session has expired. Please log in again.\n")
                        os.Exit(1)
                }
		return nil, fmt.Errorf(ret.Err)
	}
	var ents []client.DirEnt
	for _, e := range ret.Entries {
		ents = append(ents, e)
	}
	return ents, nil
}

func (c *Client) Mkdir(path string) (err error) {
	var ret string
	if path == "" {
                fmt.Print("Usage: rm <filename>\n")
		err = nil
        } else {
                err = c.server.Call("mkdir", &ret, currdir+path, user, sessionid)
        }

	if err != nil {
		return client.MakeFatalError(err)
	}
	if ret != "" {
		if(ret=="reauth"){
			fmt.Print("Your session has expired. Please log in again.\n")
                        os.Exit(1)
		}
		return fmt.Errorf(ret)
	}
	return nil
}

func (c *Client) Remove(path string) (err error) {
	var ret string
	 if path == "" {
		fmt.Print("Usage: rm <filename>\n")
                err = nil
        } else {
                err = c.server.Call("remove", &ret, currdir+path, user, sessionid)
        }

	if err != nil {
		return client.MakeFatalError(err)
	}
	if ret != "" {
		if(ret=="reauth"){
                        fmt.Print("Your session has expired. Please log in again.\n")
                        os.Exit(1)
                }
		return fmt.Errorf(ret)
	}
	return nil
}

func (c *Client) PWD() (path string, err error) {
	var ret internal.PWDReturn
	// don't actually have any information in this return value
	err = c.server.Call("pwd", &ret, user, sessionid)
	if err != nil {
		return "", client.MakeFatalError(err)
	}
	if ret.Err != "" {
		if(ret.Err=="reauth"){
                        fmt.Print("Your session has expired. Please log in again.\n")
                        os.Exit(1)
                }
		return "", fmt.Errorf(ret.Err)
	}
	return strings.TrimPrefix(currdir, "./userfs"), nil
}

func (c *Client) CD(path string) (err error) {

	var ret string
        err = c.server.Call("cd", &ret, currdir+path, user, sessionid)
        if err != nil {
                return client.MakeFatalError(err)
        }

        if ret != "" {
                if(ret == "reauth"){
                        fmt.Print("Your session has expired. Please log in again.\n")
                        os.Exit(1)
                }

                if(strings.HasPrefix(ret, "./")){
                        currdir = ret
                }else{
                        fmt.Print(ret)
                }
        }

	return nil
}
