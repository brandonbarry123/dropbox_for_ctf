package client

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

// RunCLI accepts an already-authenticated Client, and runs a command-line
// interface for the user, allowing the user to interact with the Client.
//
// If any error returned implements the FatalError interface, and IsFatal
// returns true for that error, RunCLI will return that error immediately.
// Otherwise, the error will be logged, but the client will continue running.
func RunCLI(c Client) error {
	s := bufio.NewScanner(os.Stdin)

	for {
		pwd, err := c.PWD()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error retrieving pwd: %v\n", err)
			if isFatal(err) {
				fmt.Fprintln(os.Stderr, "fatal error; aborting")
				return err
			}
		}
		fmt.Printf("%s> ", pwd)
		if !s.Scan() {
			break
		}
		parts := strings.Fields(s.Text())
		if len(parts) == 0 {
			continue
		}
		args := parts[1:]
		switch parts[0] {
		case "quit", "exit":
			if len(args) != 0 {
				fmt.Printf("Usage: %v\n", parts[0])
				break
			}
			return nil
		case "cd":
			if len(args) != 0 && len(args) != 1 {
				fmt.Printf("Usage: %v [<path>]\n", parts[0])
				break
			}
			path := "/"
			if len(args) == 1 {
				path = args[0]
			}

			err := c.CD(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error cd'ing: %v\n", err)
				if isFatal(err) {
					return err
				}
				break
			}
		case "pwd":
			if len(args) != 0 {
				fmt.Printf("Usage: %v\n", parts[0])
				break
			}
			pwd, err := c.PWD()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error getting pwd: %v\n", err)
				if isFatal(err) {
					return err
				}
				break
			}
			fmt.Println(pwd)
		case "ls":
			if len(args) != 0 && len(args) != 1 {
				fmt.Printf("Usage: %v [<path>]\n", parts[0])
				break
			}
			path := "."
			if len(args) == 1 {
				path = args[0]
			}
			ents, err := c.List(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error listing: %v\n", err)
				if isFatal(err) {
					fmt.Fprintln(os.Stderr, "fatal error; aborting")
					return err
				}
				break
			}
			for _, e := range ents {
				fmt.Println(DirEntString(e))
			}
		case "upload":
			if len(args) != 2 {
				fmt.Printf("Usage: %v <localpath> <remotepath>\n", parts[0])
				break
			}
			body, err := ioutil.ReadFile(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "error reading file: %v\n", err)
				break
			}

			err = c.Upload(args[1], body)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error uploading: %v\n", err)
				if isFatal(err) {
					return err
				}
				break
			}
		case "mkdir":
			if len(args) != 1 {
				fmt.Printf("Usage: %v <path>\n", parts[0])
				break
			}
			err := c.Mkdir(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "error making directory: %v\n", err)
				if isFatal(err) {
					return err
				}
				break
			}
		case "download":
			if len(args) != 2 {
				fmt.Printf("Usage: %v <remotepath> <localpath>\n", parts[0])
				break
			}
			body, err := c.Download(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "error downloading: %v\n", err)
				if isFatal(err) {
					return err
				}
				break
			}

			err = ioutil.WriteFile(args[1], body, 0664)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error writing file: %v\n", err)
				break
			}
		case "cat":
			if len(args) != 1 {
				fmt.Printf("Usage: %v <remotepath>\n", parts[0])
				break
			}
			body, err := c.Download(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "error downloading: %v\n", err)
				if isFatal(err) {
					return err
				}
				break
			}

			os.Stdout.Write(body)
		case "rm":
			if len(args) != 1 {
				fmt.Printf("Usage: %v <path>\n")
				break
			}
			err := c.Remove(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "error removing: %v\n", err)
				if isFatal(err) {
					return err
				}
				break
			}
		case "help":
			if len(args) != 0 {
				fmt.Printf("Usage: %v\n", parts[0])
				break
			}
			fmt.Println("Available commands:")
			cmds := []string{
				"cd [<path>]",
				"pwd",
				"ls [<path>]",
				"mkdir <path>",
				"upload <localpath> <remotepath>",
				"download <remotepath> <localpath>",
				"cat <remotepath>",
				"rm <path>",
				"quit",
				"exit",
				"help",
			}
			for _, c := range cmds {
				fmt.Println("\t" + c)
			}
		default:
			fmt.Println("Unknown command; try \"help\"")
		}
	}

	// Add a newline after the default prompt
	fmt.Println()
	if err := s.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "error scanning stdin: %v\n", err)
		return err
	}
	return nil
}

func isFatal(err error) bool {
	if f, ok := err.(FatalError); ok {
		return f.IsFatal()
	}
	return false
}
