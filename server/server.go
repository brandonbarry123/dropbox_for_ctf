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
	listenAddr := os.Args[1]
	

	rpc.RegisterHandler("unshare", unshareHandler)
	rpc.RegisterHandler("chperm", chpermHandler)
	rpc.RegisterHandler("share", shareHandler)	
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


//passing in full path
func sharerUpload(sharer string, origpath string, body []byte) string {
		stmt, err := db.Prepare("SELECT shareepath FROM sharedata where sharer=? AND origpath=?")
        if err != nil {
                fmt.Fprintf(os.Stderr, "could not make prepared statement: %v\n", err)
                os.Exit(1)
        }
        rows, err := stmt.Query(sharer, origpath)
        if err != nil {
                fmt.Fprintf(os.Stderr, "could not access database: %v\n", err)
                os.Exit(1)
        }

      	var sharee_list []string
		//removing all symlinks
        defer rows.Close()
        for rows.Next() {

            var shareepath string
            err = rows.Scan(&shareepath)
            if err != nil {
                fmt.Fprintf(os.Stderr, "could not access database: %v\n", err)
                os.Exit(1)
            }
	        //symlink code.
            sharee_list = append(sharee_list, shareepath)
            err = os.Remove(shareepath)
            if err != nil {
                return "Error removing from sharee"
            }
	
        
        }
	    uploadHelper(origpath, sharer, body)

        realfile, err := os.Readlink(origpath)
        if err != nil {
            return "Something went wrong and we couldn't access your file\n"               
        }

		size := len(sharee_list)
		for i:=0;i<size;i+=1 {
			shareepath:=sharee_list[i]
			err = os.Symlink(realfile, shareepath)
		}
        return "sharerUpload complete"
}



func getPerms(path string, sharee string) int {
        //make prepare statement to prevent sql injection
        stmt, err := db.Prepare("SELECT perm FROM sharedata WHERE shareepath=? AND sharee=?")
        if err != nil {
              fmt.Fprintf(os.Stderr, "could not make prepared statement: %v\n", err)
              os.Exit(1)
        }

        //make query for the username and password in the database
        var perm int
        err = stmt.QueryRow(path, sharee).Scan(&perm)
        if err != nil {
              fmt.Fprintf(os.Stderr, "could not make query: %v\n", err)
              os.Exit(1)
        }

        return perm

}


func isSharedFile(path string) string {
        stmt, err1 := db.Prepare("SELECT sharer FROM sharedata WHERE origpath=?")
        if err1 != nil {
              fmt.Fprintf(os.Stderr, "could not make prepared statement: %v\n", err1)
              os.Exit(1)
        }
        var foundsharer string
        err1 = stmt.QueryRow(path).Scan(&foundsharer)

        stmt, err2 := db.Prepare("SELECT sharee FROM sharedata WHERE shareepath=?")
        if err2 != nil {
              fmt.Fprintf(os.Stderr, "could not make prepared statement: %v\n", err2)
              os.Exit(1)
        }
        var foundsharee string
        err2 = stmt.QueryRow(path).Scan(&foundsharee)

        if(err1!=sql.ErrNoRows){
                return "sharer"
        }else if(err2!=sql.ErrNoRows){
                return "sharee"
        }else{
                return ""
        }
}


func uploadHelper(storepath string, username string, body []byte) string {
        prefix, err:=filepath.Abs("./userfs/"+username+"/Shared_with_me")
        if(err!=nil){
            return "Error finding path..."
        }
        if(strings.HasPrefix(storepath,prefix)){
            return "You cannot upload a new file to Shared_with_me"
        }   
        if _, err := os.Stat(storepath); err == nil {


            remove(storepath, username)

        }

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

                err = os.Symlink(abspath, storepath)

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
                _, err = result.RowsAffected()

                if err != nil {
                        fmt.Fprintf(os.Stderr, "could not access affected parts of database: %v\n", err)
                        os.Exit(1)
                }

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
                        abspath, err := filepath.Abs("./filestore/" + found)
            if err != nil {
                    return err.Error()
            }
            err = os.Symlink(abspath, storepath)

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
            _, err = result.RowsAffected()
            if err != nil {
                    fmt.Fprintf(os.Stderr, "could not access affected parts of database: %v\n", err)
                    os.Exit(1)
            }
        }
        return "End of uploadHelper"
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
	_, err = result.RowsAffected()
	if err != nil {
              fmt.Fprintf(os.Stderr, "could not fetch username and password: %v\n", err)
              os.Exit(1)
        }


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






func chpermHandler(path string, sharee string, newperm string, username string, cookie string) string {
	if(checkCookie(username, cookie)==false){
                return "reauth"
        }
	
	    allow := checkpath(path, username)
    if(allow==true){

        owner_shared, err := filepath.Abs("./userfs/" + username + "/Shared_with_me") 
        if err != nil {
            return "Oops, abs failed!"
        }

        fullpath, err := filepath.Abs(filepath.Clean(path))
        if(err!=nil){
                fmt.Fprintf(os.Stderr, "error abs path: %v\n", err)
                return ""
        }

        if strings.HasPrefix(fullpath, owner_shared){
            return "Dude, you don't own this!"
        }

        if _, err := os.Stat(fullpath); os.IsNotExist(err) {
                return "That resource doesn't exist!\n"
        }

        stmt, err := db.Prepare("SELECT count(1) FROM sharedata WHERE sharer=? AND sharee=? AND origpath=?")
        if err != nil {
            fmt.Fprintf(os.Stderr, "could not make prepared statement: %v\n", err)
            os.Exit(1)
        }

        var found int
        err = stmt.QueryRow(username, sharee, fullpath).Scan(&found)
        if err != nil {
            fmt.Fprintf(os.Stderr, "could not make query: %v\n", err)
            os.Exit(1)
        }

        if found == 0 {
            return "File not shared with this person."
        }

        stmt, err = db.Prepare("UPDATE sharedata SET perm=? WHERE sharer=? AND sharee=? AND origpath=?")
        if err != nil {
              fmt.Fprintf(os.Stderr, "could not make prepared statement: %v\n", err)
              os.Exit(1)
        }
        
        perm := 0
        if newperm == "rw"{
            perm = 1
        } else { 
            if newperm != "r" {
                return "Permissions can only be r or rw"
            }
        }


        _, err = stmt.Exec(perm, username, sharee, fullpath)
        if err != nil {
              fmt.Fprintf(os.Stderr, "could not make query: %v\n", err)
              os.Exit(1)
        }

        return "Permissions updated"
    } else {
        return "You don't have access to this."
    }  



	return "chperm"





}





func shareHandler(path string, sharee string, permissions string, username string, cookie string) string {
	if(checkCookie(username, cookie)==false){
                return "reauth"
    }
    allow := checkpath(path, username)
    if(allow==true){
    	if username == sharee {
    		return "You can't share it with yourself, silly!"
    	}
    	perm := 0
    	if permissions == "rw" {
    		perm = 1
    	} else {
    		if permissions != "r"{
    			return "Permissions can only be either r or rw\n"
    		}
    	}

    	stmt, err := db.Prepare("SELECT count(1) FROM userdata WHERE username=?")
            if err != nil {
                  fmt.Fprintf(os.Stderr, "could not make prepared statement: %v\n", err)
                  os.Exit(1)
            }
            var found int
            err = stmt.QueryRow(sharee).Scan(&found)
            if err != nil {
                  fmt.Fprintf(os.Stderr, "could not make query: %v\n", err)
                  os.Exit(1)
            }
            if(found == 0){
                    return "The user you're trying to share with doesn't exist!\n" 
            }

    	fullpath, err := filepath.Abs(filepath.Clean(path))
        if(err!=nil){
                fmt.Fprintf(os.Stderr, "error abs path: %v\n", err)
                return ""
        }

        if _, err := os.Stat(fullpath); os.IsNotExist(err) {
                return "That resource doesn't exist!\n"
        }
    	owner_shared, err := filepath.Abs("./userfs/" + username + "/Shared_with_me") 
    	if err != nil {
    		return "Oops, abs failed!"
    	}
    	if strings.HasPrefix(fullpath, owner_shared){
    		return "Dude, you don't own this!"
    	}

    	stmt, err = db.Prepare("SELECT count(1) FROM sharedata WHERE sharer=? AND sharee=? AND origpath=?")
            if err != nil {
                  fmt.Fprintf(os.Stderr, "could not make prepared statement: %v\n", err)
                  os.Exit(1)
            }


        err = stmt.QueryRow(username, sharee, fullpath).Scan(&found)


    	if err != nil {
                  fmt.Fprintf(os.Stderr, "could not make query: %v\n", err)
                  os.Exit(1)
            }
            if(found == 1){
                    return "You already shared this with this user! If you want to change permissions, use chperm.\n"
            }


           
    	//at this point, auth, checked file, checked username 

    	path_to_sharee, err := filepath.Abs("./userfs/"+sharee+"/Shared_with_me/")
    	if(err!=nil){
                    fmt.Fprintf(os.Stderr, "error abs path: %v\n", err)
                    return "Something went wrong :(\n"
            }
    	filename := filepath.Base(fullpath)
    	if(err!=nil){
                    fmt.Fprintf(os.Stderr, "error abs path: %v\n", err)
                    return "Something went wrong :(\n"
            }
    	i := 1
    	
    	_, err = os.Stat(path_to_sharee +"/"+ filename)
    	

    	tmpfile := filename
    	loopvar := os.IsNotExist(err)
    	for !loopvar {
    		tmpfile=filename

    		tmpfile = strconv.Itoa(i) + tmpfile

    		_, err = os.Stat(path_to_sharee +"/"+ tmpfile); 	
    		loopvar = os.IsNotExist(err)
    		i += 1
    	}
    	filename = tmpfile


     
    	///////////////////////////////////////

    	filedata, err := os.Lstat(fullpath)
        if err != nil {
             return "This is not a file or we could not locate it!\n"
        }

        if filedata.Mode()&os.ModeSymlink != 0 {
            newpath, err := os.Readlink(fullpath)
            if err != nil {
 		         return "Something went wrong and we couldn't access your file\n"               
            }

	       err = os.Symlink(newpath, path_to_sharee + "/" + filename)    
           if err != nil {
		    fmt.Print(err.Error())
                    return "Could not share!"
            }
	       stmt, err = db.Prepare("INSERT INTO sharedata(sharer, sharee, origpath, shareepath, perm) values(?,?,?,?,?)")
           if err != nil {
                    fmt.Fprintf(os.Stderr, "could not make prepared statement: %v\n", err)
                    os.Exit(1)
           }
	       _, err = stmt.Exec(username, sharee, fullpath, path_to_sharee + "/" + filename, perm)
            if err != nil {
                    fmt.Fprintf(os.Stderr, "could not update database: %v\n", err)
                    os.Exit(1)
            }    
            return "Shared!"
    		
    	}	
         
    	return "There seems to have been an issue"
	} else {
        return "You don't have access to this resource"
    }



}



func unshareHandler(path string, sharee string, username string, cookie string) string {
	if(checkCookie(username, cookie)==false){
                return "reauth"
        }
    allow := checkpath(path, username)
	if(allow==true){
        if username == sharee {
            return "You can't share it with yourself, silly!"
        }

        stmt, err := db.Prepare("SELECT count(1) FROM userdata WHERE username=?")
        if err != nil {
            fmt.Fprintf(os.Stderr, "could not make prepared statement: %v\n", err)
            os.Exit(1)
        }
        var found int
        err = stmt.QueryRow(sharee).Scan(&found)
        if err != nil {
            fmt.Fprintf(os.Stderr, "could not make query: %v\n", err)
            os.Exit(1)
        }
        if(found == 0){
            return "The user you're trying to unshare with doesn't exist!\n" 
        }

        fullpath, err := filepath.Abs(filepath.Clean(path))
        if(err!=nil){
                fmt.Fprintf(os.Stderr, "error abs path: %v\n", err)
                return ""
        }


        stmt, err = db.Prepare("SELECT count(1) FROM sharedata WHERE sharer=? AND sharee=? AND origpath=?")
        if err != nil {
            fmt.Fprintf(os.Stderr, "could not make prepared statement: %v\n", err)
            os.Exit(1)
        }

        err = stmt.QueryRow(username, sharee, fullpath).Scan(&found)


        if err != nil {
            fmt.Fprintf(os.Stderr, "could not make query: %v\n", err)
            os.Exit(1)
        }

        if(found != 1){
            return "You have not shared this file with the specified user.\n"
        }

        stmt, err = db.Prepare("SELECT shareepath FROM sharedata WHERE sharer=? AND sharee=? AND origpath=?")
        if err != nil {
            fmt.Fprintf(os.Stderr, "could not make prepared statement: %v\n", err)
            os.Exit(1)
        }
        var sym_to_remove string
        err = stmt.QueryRow(username, sharee, fullpath).Scan(&sym_to_remove)

        err = os.Remove(sym_to_remove)
        if err != nil {
            return err.Error()
        }   

        stmt, err = db.Prepare("DELETE FROM sharedata WHERE sharer=? AND sharee=? AND origpath=?")
        if err != nil {
                fmt.Fprintf(os.Stderr, "could not make prepared statement: %v\n", err)
                os.Exit(1)
        }

        _, err = stmt.Exec(username, sharee, fullpath)

        if err != nil {
                fmt.Fprintf(os.Stderr, "could not make prepared statement: %v\n", err)
                os.Exit(1)
        }

    	return "unshared!"
    } else {
        return "You do not have access to this resource"
    }


}


func uploadHandler(path, username string, body []byte, cookie string) string {
        if(checkCookie(username, cookie)==false){
                return "reauth"
        }

        allow := checkpath(path, username)


         if(allow==true){
                storepath, err := filepath.Abs(path)
                if err != nil {
                        return err.Error()
                }

                shared := isSharedFile(storepath)

                if(shared==""){
                    return uploadHelper(storepath, username, body)
                }else{
                        //case the file is shared
                        if(shared=="sharer"){
                                //sharerupload

			    return sharerUpload(username, storepath, body)

                        }else{
                                // need to change this to have one more argument

                                perms:=getPerms(storepath, username)
                                if(perms==0){
                                        return "Permission Denied"

                                }else{
                                       	//get sharer and pass in 
					stmt, err := db.Prepare("SELECT sharer, origpath FROM sharedata WHERE shareepath=?")
        				if err != nil {
              					fmt.Fprintf(os.Stderr, "could not make prepared statement: %v\n", err)
              					os.Exit(1)
        				}
					var sharerpath string
       					var foundsharer string
        				err = stmt.QueryRow(storepath).Scan(&foundsharer, &sharerpath)
					if err != nil {
                                                fmt.Fprintf(os.Stderr, "could not make query: %v\n", err)
                                                os.Exit(1)
                                        }					
                                        sharerUpload(foundsharer, sharerpath, body)
                                        return "File Shared!"
                                }


                        }


                        return shared
                }
        }else{
                return "Path does not exist on the server!"
        }


        return ""


}






// An implementation of a basic server. This implementation
// is absurdly insecure, and is only meant as an example of
// how to implement the methods required by the example client
// provided in client/client.go; it should not be taken as
// a suggestion of how to design your server.



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

func remove(path string, username string) string {

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
                _, err = result.RowsAffected()
                if err != nil {
                    fmt.Fprintf(os.Stderr, "could not access affected parts of database: %v\n", err)
                    os.Exit(1)
                }
   
            } else {
                err = os.Remove(newpath)
                if err != nil {
                    return err.Error()
                }
                stmt, err = db.Prepare("DELETE FROM filedata WHERE filename=?")
                if err != nil {
                        fmt.Fprintf(os.Stderr, "could not make prepared statement: %v\n", err)
                        os.Exit(1)
                }
                _, err := stmt.Exec(origin_name)
                if err != nil {
                        fmt.Fprintf(os.Stderr, "could not update database: %v\n", err)
                        os.Exit(1)
                }

            }
		   
            return ""
        } else {
		notallow,err := filepath.Abs("./userfs/" + username + "/Shared_with_me")
		if err != nil {
                        return err.Error()
                }
		if(abspath!=notallow){
			err = os.Remove(abspath)
			if err != nil {
                		return "That directory isn't empty!\n"
            		}	
		}else{
			return "You can't remove your Shared directory!\n"
		}
        }   
    }else{
        return "You can't go outside of your directory!\n"
    }
    return ""

}



func removeHandler(path string, username string, cookie string) string {
    if(checkCookie(username, cookie)==false){
        return "reauth"
    }

    allow := checkpath(path, username)
    // If the user is allowed access to the path they have mentioned:
    if(allow==true){
        fullpath, err := filepath.Abs(filepath.Clean(path))
        if(err!=nil){
                fmt.Fprintf(os.Stderr, "error abs path: %v\n", err)
                return ""
        }

        if _, err := os.Stat(fullpath); os.IsNotExist(err) {
                return "That resource doesn't exist!\n"
        }

        shared := isSharedFile(fullpath)

        if shared == "" {
            return remove(path, username)
        } else{
            if shared == "sharee" {
                err = os.Remove(fullpath)
                if err != nil {
                    return err.Error()
                }  
                stmt, err := db.Prepare("DELETE FROM sharedata WHERE sharee=? AND shareepath=?")
                if err != nil {
                        fmt.Fprintf(os.Stderr, "could not make prepared statement: %v\n", err)
                        os.Exit(1)
                }
                _, err = stmt.Exec(username, fullpath)
                if err != nil {
                        fmt.Fprintf(os.Stderr, "could not update database: %v\n", err)
                        os.Exit(1)
                } 

            } else {
                stmt, err := db.Prepare("SELECT shareepath FROM sharedata where sharer=? AND origpath=?")
                if err != nil {
                        fmt.Fprintf(os.Stderr, "could not make prepared statement: %v\n", err)
                        os.Exit(1)
                }
                rows, err := stmt.Query(username, fullpath)
                if err != nil {
                        fmt.Fprintf(os.Stderr, "could not access database: %v\n", err)
                        os.Exit(1)
                }
                var sharee_list []string
                defer rows.Close()
                for rows.Next() {
                    var shareepath string
                    err = rows.Scan(&shareepath)
                    if err != nil {
                        fmt.Fprintf(os.Stderr, "could not access database: %v\n", err)
                        os.Exit(1)
                    }

                    
                    sharee_list = append(sharee_list, shareepath)
                }

                stmt, err = db.Prepare("DELETE FROM sharedata where sharer=? AND origpath=? AND shareepath=?")
                if err != nil {
                        fmt.Fprintf(os.Stderr, "could not make prepared statement: %v\n", err)
                        os.Exit(1)
                }
                defer rows.Close()
                
                size := len(sharee_list)
                for i := 0; i < size; i += 1 {
                    shareepath := sharee_list[i]
                    

                    err = os.Remove(shareepath)
                    if err != nil {
                        fmt.Println(err)
                        return "Could not unshare with someone"
                    }	
                    _, err = stmt.Exec(username, fullpath, shareepath)
                    if err != nil {
                        fmt.Fprintf(os.Stderr, "could not access database: %v\n", err)
                        os.Exit(1)
                    }
                    
                }
                return remove(path, username)

            }

         }   

    } else {
        return "This isn't something in your directory!"
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
