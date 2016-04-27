package main

import (
	"fmt"
	"io/ioutil"
	"encoding/base64"
	"os"
	"crypto/sha1"	
	"database/sql"
	_ "github.com/mattn/go-sqlite3"			
	"crypto/rand"
	"time"
	"path/filepath"
	"../internal"
	"../lib/support/rpc"
	"strings"
	"strconv"
)


//Cookie Struct
type Cookie struct {
        sessionid string
        expiretime time.Time
}

//Global variables
var db *sql.DB
var Cookiemap = make(map[string]Cookie)
var filecount int

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %v <listen-address>\n", os.Args[0])
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Database Initialized...\n")
	var err error
	db, err = sql.Open("sqlite3", "./dropbox.db")
	if(err!=nil){
                fmt.Fprintf(os.Stderr, "could not run server: %v\n", err)
                os.Exit(1)
        }
	filecount_read, err := ioutil.ReadFile("./filecount.txt")	
	if(err!=nil){
		fmt.Fprintf(os.Stderr, "could not read filecount: %v\n", err)
                os.Exit(1)		
	}
	filecount, err = strconv.Atoi(strings.TrimSuffix(string(filecount_read), "\n"))
	if(err!=nil){
                fmt.Fprintf(os.Stderr, "Atoi fail: %v\n", err)
                os.Exit(1)
        }
	fmt.Print(filecount)
	listenAddr := os.Args[1]

	
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
	err = rpc.RunServer(listenAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not run server: %v\n", err)
		os.Exit(1)
	}
}


func checkpath(path string, username string) bool{
	basepath, err := filepath.Abs(filepath.Clean("./userfs/" + username))
	if(err!=nil){
                fmt.Fprintf(os.Stderr, "abs broke: %v\n", err)
        }   
	desiredpath, err := filepath.Abs(filepath.Clean(path))
	
	if(err!=nil){
		fmt.Fprintf(os.Stderr, "abs broke: %v\n", err)
	}	
	if(strings.HasPrefix(desiredpath, basepath)){
		return true
	}else{
		return false
	}
}


func checkCookie(username string, session string) bool {
        fetchedcookie := Cookiemap[username]
        expirytime := fetchedcookie.expiretime
        if(expirytime.After(time.Now())){
                if(fetchedcookie.sessionid == session){
                        return true
                }
        }
        return false
}




//Handler to handle authentication requests made by the client only when the user is attempting to sign in
func authenticateHandler(username string, password string) internal.AuthReturn{	
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
		//make new random cookie
		rb:=make([]byte, 64)
                _, err := rand.Read(rb)
                if err != nil {
                        fmt.Println(err)
                }
		//store cookie and return to client
                newsession:=base64.URLEncoding.EncodeToString(rb)
                exptime := time.Now().Add(time.Second*1000)
                newcookie:=Cookie{newsession, exptime}
                Cookiemap[username] = newcookie
                return internal.AuthReturn{Auth: true, Session: newsession}	
	}else{
		return internal.AuthReturn{Auth:false, Session: ""}
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

	path := "./userfs/" + username  
	err = os.Mkdir(path, 0775)
	if err != nil {
              fmt.Fprintf(os.Stderr, "could not make directory: %v\n", err)
              os.Exit(1)	
        }

	err = os.Mkdir(path+"/Shared_with_me", 0775)
        if err != nil {
              fmt.Fprintf(os.Stderr, "could not make directory: %v\n", err)
              os.Exit(1)        
        }
	return true

}





// An implementation of a basic server. This implementation
// is absurdly insecure, and is only meant as an example of
// how to implement the methods required by the example client
// provided in client/client.go; it should not be taken as
// a suggestion of how to design your server.

func uploadHandler(path, username string, body []byte, cookie string) string {
       	if(checkCookie(username, cookie)==false){
        	return "reauth"
        }

	allow := checkpath(path, username)

	
	 if(allow==true){
		//dedup
		h := sha1.New()
       	h.Write(body)
        hash := base64.URLEncoding.EncodeToString(h.Sum(nil))
		

		
		//make prepare statement to prevent sql injection
    	stmt, err := db.Prepare("SELECT count(1) FROM filedata WHERE filehash=?")
    	if err != nil {
         	 	fmt.Fprintf(os.Stderr, "could not make prepared statement: %v\n", err)
          		os.Exit(1)
    	}

    	//make query for the username and password in the database
    	var found int
    	err = stmt.QueryRow(hash).Scan(&found)
    	if err != nil {
          		fmt.Fprintf(os.Stderr, "could not make query: %v\n", err)
          		os.Exit(1)
   		}	
		if(found==0){	

            store_at := "./filestore/file" + strconv.Itoa(filecount)
 
        	abspath, err := filepath.Abs(store_at)
            if err != nil {
                    return err.Error()
            }

            err = ioutil.WriteFile(abspath, body, 0664)

            if err != nil {
                    return err.Error()
            }

            err = os.Symlink(abspath, path)

            if err != nil {
                    return err.Error()
            }
            
            stmt, err = db.Prepare("INSERT INTO filedata(filename, filehash, numowners) values(?,?,?)")
            if err != nil {
                    fmt.Fprintf(os.Stderr, "could not make prepared statement: %v\n", err)
                    os.Exit(1)
            }
            result, err := stmt.Exec("file" + strconv.Itoa(filecount), hash, 1)
            if err != nil {
                    fmt.Fprintf(os.Stderr, "could not update database: %v\n", err)
                    os.Exit(1)
            }
            affect, err := result.RowsAffected()
            if err != nil {
                    fmt.Fprintf(os.Stderr, "could not access affected parts of database: %v\n", err)
                    os.Exit(1)
            }
            fmt.Println(affect)   

        
            filecount += 1
        	return ""

		}else{
            stmt, err = db.Prepare("SELECT filename FROM filedata WHERE filehash=?")
            if err != nil {
                    fmt.Fprintf(os.Stderr, "could not make prepared statement: %v\n", err)
                    os.Exit(1)
            }
            var found string
            err = stmt.QueryRow(hash).Scan(&found)
            if err != nil {
                    fmt.Fprintf(os.Stderr, "could not make query: %v\n", err)
                    os.Exit(1)
            }   
			abspath, err := filepath.Abs(found)
            if err != nil {
                    return err.Error()
            }
            err = os.Symlink(abspath, path)

            if err != nil {
                    return err.Error()
            }				

            stmt, err = db.Prepare("SELECT numowners FROM filedata WHERE filehash=?")
            if err != nil {
                    fmt.Fprintf(os.Stderr, "could not make prepared statement: %v\n", err)
                    os.Exit(1)
            }
            var curr_num int
            err = stmt.QueryRow(hash).Scan(&curr_num)

            new_num := curr_num + 1

            stmt, err = db.Prepare("UPDATE filedata SET numowners=? WHERE filehash=?")
            if err != nil {
                    fmt.Fprintf(os.Stderr, "could not make prepared statement: %v\n", err)
                    os.Exit(1)
            }
            result, err := stmt.Exec(new_num, hash)
            if err != nil {
                    fmt.Fprintf(os.Stderr, "could not update database: %v\n", err)
                    os.Exit(1)
            }
            affect, err := result.RowsAffected()
            if err != nil {
                    fmt.Fprintf(os.Stderr, "could not access affected parts of database: %v\n", err)
                    os.Exit(1)
            }
            fmt.Println(affect)   

		}

          
        }else{ 
                return "Path does not exist on the server!"
        }

	
        return ""
	

}

func downloadHandler(path, username, cookie string) internal.DownloadReturn {
	if(checkCookie(username, cookie)==false){
        	return internal.DownloadReturn{Err: "reauth"}
    	}
	
	allow := checkpath(path, username)
    	if(allow==true){
        	abspath, err := filepath.Abs(path)
        	if err != nil {
           		return internal.DownloadReturn{Err: err.Error()}
        	}
    		filedata, err := os.Lstat(abspath)
        	if err != nil {
                	return internal.DownloadReturn{Err: err.Error()}
        	}  

        if filedata.Mode()&os.ModeSymlink != 0 {
            newpath, err := os.Readlink(abspath)
            if err != nil {
                return internal.DownloadReturn{Err: err.Error()}
            }  
            body, err := ioutil.ReadFile(newpath)
            if err != nil {
                return internal.DownloadReturn{Err: err.Error()}
            }   
            return internal.DownloadReturn{Body: body}
        } else {
            if err != nil {
                    return internal.DownloadReturn{Err: "Invalid file :(\n"}
            }  
        }

    } 
    return internal.DownloadReturn{Err: "Path does not exist!"}
}

func listHandler(path, username, cookie string) internal.ListReturn {
	if(checkCookie(username, cookie)==false){
                return internal.ListReturn{Err: "reauth"}
        }

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

func mkdirHandler(path string, username string, cookie string) string {
	if(checkCookie(username, cookie)==false){
                 return "reauth"
        }
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

func removeHandler(path string, username string, cookie string) string {
    if(checkCookie(username, cookie)==false){
        return "reauth"
    }

	allow := checkpath(path, username)
    // If the user is allowed access to the path they have mentioned:
    if(allow==true){
        abspath, err := filepath.Abs(path)
        if err != nil {
            return err.Error()
        }
        filedata, err := os.Lstat(abspath)
        if err != nil {
                return err.Error()
        }  
        // If that path is a legitimate symbolic link in their directory
        if filedata.Mode()&os.ModeSymlink != 0 {
            // Get the file the path links to
            newpath, err := os.Readlink(abspath)
            if err != nil {
                return err.Error()
            }  
            err = os.Remove(abspath)
            if err != nil {
                return err.Error()
            }   
            // Get the name of the origin file and the number of users who have access to the file before deletion
            parts := strings.Split(newpath, "/")
            origin_name := parts[len(parts) - 1]
            stmt, err := db.Prepare("SELECT numowners FROM filedata WHERE filename=?")
            if err != nil {
                    fmt.Fprintf(os.Stderr, "could not make prepared statement: %v\n", err)
                    os.Exit(1)
            }
            var curr_num int
            err = stmt.QueryRow(origin_name).Scan(&curr_num)

            new_num := curr_num - 1

            if new_num > 0 {
                stmt, err = db.Prepare("UPDATE filedata SET numowners=? WHERE filename=?")
                if err != nil {
                        fmt.Fprintf(os.Stderr, "could not make prepared statement: %v\n", err)
                        os.Exit(1)
                }
                result, err := stmt.Exec(new_num, origin_name)
                if err != nil {
                        fmt.Fprintf(os.Stderr, "could not update database: %v\n", err)
                        os.Exit(1)
                }
                affect, err := result.RowsAffected()
                if err != nil {
                    fmt.Fprintf(os.Stderr, "could not access affected parts of database: %v\n", err)
                    os.Exit(1)
                }
                fmt.Println(affect)   
            } else {
                err = os.Remove(newpath)
                if err != nil {
                    return err.Error()
                }
                stmt, err = db.Prepare("DELETE FROM filedata WHERE filename=origin_name")
                if err != nil {
                        fmt.Fprintf(os.Stderr, "could not make prepared statement: %v\n", err)
                        os.Exit(1)
                }
                result, err := stmt.Exec(origin_name)
                if err != nil {
                        fmt.Fprintf(os.Stderr, "could not update database: %v\n", err)
                        os.Exit(1)
                }
                affect, err := result.RowsAffected()
                if err != nil {
                        fmt.Fprintf(os.Stderr, "could not access affected parts of database: %v\n", err)
                        os.Exit(1)
                }
                fmt.Println(affect)

            }   
            return ""
        } else {
            if err != nil {
                return "This doesn't seem to be a file you saved around these parts!\n"
            }  
        }   
    }else{
        return "You can't go outside of your directory!\n"
    }
    return ""

}

func pwdHandler(username string, cookie string) internal.PWDReturn {
	if(checkCookie(username, cookie)==false){
                return internal.PWDReturn{Err: "reauth"}
        }




	path, err := os.Getwd()
	if err != nil {
		return internal.PWDReturn{Err: err.Error()}
	}
	return internal.PWDReturn{Path: path}
}

func cdHandler(path string, username string, cookie string) string {
	if(checkCookie(username, cookie)==false){
                return "reauth"
        }





	//path is relative to current path.... should be in home directory. Lets not make it hardcoded...

	
	allow := checkpath(path, username)
	if(allow==true){	
		//err := os.Chdir(path)
		desiredpath, err := filepath.Abs(filepath.Clean(path))
		if(err!=nil){
                        fmt.Fprintf(os.Stderr, "error abs path: %v\n", err)
			return ""
                }
                
                if _, err := os.Stat(desiredpath); os.IsNotExist(err) {
                         return "That resource doesn't exist!\n"
                }
		totrim, err := filepath.Abs("./userfs")
			
		if(err!=nil){
                        fmt.Fprintf(os.Stderr, "error abs path: %v\n", err)
                        return ""
                }
		
		displaydir := strings.TrimPrefix(desiredpath, totrim)
		return "./userfs" + displaydir +"/"

	}else{
		return "You can't go outside of your directory!\n"
	}

	return ""
}

func finalizer() {
    ioutil.WriteFile("./filecount.txt", []byte(strconv.Itoa(filecount)), 0664)
	fmt.Println("Shutting down...")
}
