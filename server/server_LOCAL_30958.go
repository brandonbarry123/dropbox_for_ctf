package main

import (
	"fmt"
	"io/ioutil"
	"encoding/base64"
	"os"
	"crypto/sha1"	
	"database/sql"
	_ "github.com/mattn/go-sqlite3"			
//	"github.com/gorilla/sessions"
	"path/filepath"
	"../internal"
	"../lib/support/rpc"
	"strings"
)


//Global database variable	
var db *sql.DB
//Global session store
//var store = sessions.NewCookieStore([]bytes("password"))


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
	//opens database


	fmt.Fprintf(os.Stderr, "Database Initialized...\n")
	var err error
	db, err = sql.Open("sqlite3", "./../dropbox.db")
		
	if(err!=nil){
		fmt.Fprintf(os.Stderr, "could not run server: %v\n", err)
                os.Exit(1)		
	}
	
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
	rpc.RegisterHandler("authenticate", authenticateHandler)
	rpc.RegisterHandler("signup", signupHandler)
//	rpc.RegisterHandler("getcookie", getcookieHandler)
//	rpc.RegisterHandler("checkcookie", setcookieHandler)	
	err = rpc.RunServer(listenAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not run server: %v\n", err)
		os.Exit(1)
	}
}


func checkpath(path string, username string) bool{
	basepath := "/Users/harjasleen/Desktop/Brown Classes/Spring2016/CS162/s16-bjb-hmalvai/userfs/" + username
	 
	desiredpath, err := filepath.Abs(filepath.Clean(path))
	
	if(err!=nil){
		fmt.Fprintf(os.Stderr, "abs broke: %v\n", err)
	}	
	fmt.Print("basepath:" + basepath+ "\n")
	fmt.Print("desiredpath:" + desiredpath+ "\n")	
	if(strings.HasPrefix(desiredpath, basepath)){
		return true
	}else{
		return false
	}
}




func addHandler(a, b int) int  { return a + b }
func multHandler(a, b int) int { return a * b }
func noOpHandler()             {}



//Handler to handle authentication requests made by the client only when the user is attempting to sign in
func authenticateHandler(username string, password string) bool{	
	h := sha1.New()
	h.Write([]byte(password))
	hash := base64.URLEncoding.EncodeToString(h.Sum(nil))
		
	//make prepare statement to prevent sql injection
	stmt, err := db.Prepare("SELECT count(1) FROM userdata WHERE username=? AND passhash=?")
	if err != nil {
              fmt.Fprintf(os.Stderr, "could not make prepared statement: %v\n", err)
              os.Exit(1)
        }

	//make query for the username and password in the database	
	var found int
	err = stmt.QueryRow(username, hash).Scan(&found)
	if err != nil {
              fmt.Fprintf(os.Stderr, "could not make query: %v\n", err)
              os.Exit(1)
        }
	if(found == 1){
		//path:= "../userfs/" + username

		//err = os.Chdir(path)
        	//if err != nil {
               	//	fmt.Fprintf(os.Stderr, "could not change directory: %v\n", err)
        	//}
		return true	
	}else{
		return false
	}
}





func signupHandler(username string, password string) bool {
        stmt, err := db.Prepare("SELECT count(1) FROM userdata WHERE username=?")
        if err != nil {
              fmt.Fprintf(os.Stderr, "could not make prepared statement: %v\n", err)
              os.Exit(1)
        }

        //make query for the username in the database
        var found int
        err = stmt.QueryRow(username).Scan(&found)

        if err != nil {
              fmt.Fprintf(os.Stderr, "could not make query: %v\n", err)
              os.Exit(1)
        }
        if(found == 1){
		fmt.Fprintf(os.Stderr, "Username already exists!")
                return false
        }
        
	h := sha1.New()
        h.Write([]byte(password))
        hash := base64.URLEncoding.EncodeToString(h.Sum(nil))

        //make prepare statement to prevent sql injection
        stmt, err = db.Prepare("INSERT INTO userdata (username, passhash) VALUES (?, ?)")
        if err != nil {
              fmt.Fprintf(os.Stderr, "could not make prepared statement: %v\n", err)
              os.Exit(1)
        }
	result, err := stmt.Exec(username, hash)
        if err != nil {
              fmt.Fprintf(os.Stderr, "could not make prepared statement: %v\n", err)
              os.Exit(1)
        }
	fmt.Fprintf(os.Stderr, "Your account has been created")	
	affect, err := result.RowsAffected()
	if err != nil {
              fmt.Fprintf(os.Stderr, "could not fetch username and password: %v\n", err)
              os.Exit(1)
        }
	fmt.Println(affect)

	path := "/root/s16-bjb-hmalvai/userfs/" + username  
	err = os.Mkdir(path, 0775)
	if err != nil {
              fmt.Fprintf(os.Stderr, "could not make directory: %v\n", err)
              os.Exit(1)	
        }
	return true

}






//func getcookieHandler() string {
//	session, _ := store.Get(	

//	return "cookie"
//}



//func checkcookieHandler() boolean {

//	return true
//}




// An implementation of a basic server. This implementation
// is absurdly insecure, and is only meant as an example of
// how to implement the methods required by the example client
// provided in client/client.go; it should not be taken as
// a suggestion of how to design your server.

func uploadHandler(path, username string, body []byte) string {
        allow := checkpath(path, username)
        if(allow==true){ 
        	err := ioutil.WriteFile(path, body, 0664)
        	if err != nil {
                	return err.Error()
        	}   
        	return ""          
         }else{ 
                return "Path does not exist on the server!"
         }

}

func downloadHandler(path, username string) internal.DownloadReturn {
	allow := checkpath(path, username)
        if(allow==true){
        	body, err := ioutil.ReadFile(path)
        	if err != nil {
                	return internal.DownloadReturn{Err: err.Error()}
        	}   
        	return internal.DownloadReturn{Body: body}
                          
         }else{
                return internal.DownloadReturn{Err: "Path does not exist!"}
         }
}

func listHandler(path, username string) internal.ListReturn {
	allow := checkpath(path, username)
        if(allow==true){
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
        }else{
                return internal.ListReturn{Err: "Directory does not exist!"}
        }

}

func mkdirHandler(path, username string) string {
	allow := checkpath(path, username)
        if(allow==true){
        	err := os.Mkdir(path, 0775)
       		if err != nil {
                	return err.Error()
        	}
        	return ""
        }else{
                return "You can't go outside of your directory!\n"
        }
}

func removeHandler(path, username string) string {
        allow := checkpath(path, username)
        if(allow==true){
                err := os.Remove(path)
        	if err != nil {
                	return err.Error()
        	}
        	return ""
        }else{
                return "You can't go outside of your directory!\n"
        }

}

func pwdHandler() internal.PWDReturn {
	path, err := os.Getwd()
	if err != nil {
		return internal.PWDReturn{Err: err.Error()}
	}
	return internal.PWDReturn{Path: path}
}

func cdHandler(path string, username string) string {
	//path is relative to current path.... should be in home directory. Lets not make it hardcoded...

	
	allow := checkpath(path, username)
	if(allow==true){	
		//err := os.Chdir(path)
		desiredpath, err := filepath.Abs(filepath.Clean(path))
		if(err!=nil){
			fmt.Fprintf(os.Stderr, "error abs path: %v\n")
			return ""
		} else {
			return desiredpath
		}
	}else{
		return "You can't go outside of your directory!\n"
	}

	return ""
}

func finalizer() {
	fmt.Println("Shutting down...")
}
