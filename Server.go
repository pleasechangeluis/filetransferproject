package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"io"
)
type execFunc func () int
var theListener net.Listener
var clients []net.Conn
var clientsMutex sync.Mutex

func helpmenu() int{
	fmt.Println("All commmands are case sensitive, so be careful when typing them in. \nCurrent commands are the following:")
	fmt.Println("help - prints out text that lists out all commands and then describes each one.")
	fmt.Println("exit - closes the program")
	fmt.Println("viewfiles - lists out the files people will be able to download from you.")
	fmt.Println("updatefile - will ask you for a file name, increments the version value of the file.")
	fmt.Println("start - commences sever to allow connection")
	fmt.Println("list - prints all current connections to the server")
	return 1
}

func cexit() int{
	fmt.Println("clean")
	fmt.Println("exit.")
	if theListener != nil{
		theListener.Close()
	}
	return 1
}

func vfiles() int {
	entries, err := os.ReadDir("./downloadables")
	if err != nil {
		panic(err)
	}
	if len(entries) == 0 {
		fmt.Println("You have no files to share! Try dragging files into your \"downloadables\" folder.")
	} else {
		for index, entry := range entries{
			//fmt.Printf("%d \n", index+1)
			var name = entry.Name()
			fmt.Printf("%d) %v ", index+1, name)
			var location = "./downloadables/" + name + "/info.txt"
			data, err := os.ReadFile(location)
			if err != nil {
				panic(err)
			}
			var newlineloc = strings.Index(string(data), "\n")
			fmt.Printf("%.*s\n", newlineloc, data)
			newlineloc  += 1
		}
	}


	return 1
}

func star() int {
	ln, neterr := net.Listen("tcp", ":8080")
	if neterr != nil {
		panic(neterr)
	}
	go _acceptconnections(ln)
	theListener = ln
	return 1
}

func _acceptconnections(ln net.Listener) {
	for {
            conn, err := ln.Accept()
            if err != nil {
                fmt.Println("Error accepting connection:", err)
                return
            }
			// the mutex prevents the main server code from interacting with the
			// slice memory while this goroutine is about to modify it
			// this prevents a read/write error
			clientsMutex.Lock()
			clients = append(clients, conn)
			clientsMutex.Unlock()


			// replace this write with a "handle connection" goroutine.
			// this routine should manage the connection and allow the server
			// to intereact and terminate itself if the connection closes
			go _handling_client(conn)
        }
}

func _handling_client(conn net.Conn){
	defer _client_exit(conn)

	scanner := bufio.NewScanner(conn)
	fmt.Fprintln(conn, "hello welcome to this database")

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "error reading:", err)
	}

	for scanner.Scan(){
		
		text := scanner.Text()
		fmt.Printf("Command recivied: %v\n", text)

		if text == "browse"{
			client_browse(conn)
			fmt.Fprintln(conn, "done")
		}

		if text == "download"{
			scanner.Scan()
			filenumstr := scanner.Text()
			filenum, errstring := strconv.Atoi(filenumstr)
			if errstring != nil {
				fmt.Fprintln(conn, "error")
				fmt.Fprintln(conn, "done")
				return
			}
			// iterate thru avialable files to correct number is reached. send the file thru pipe
			entries, err := os.ReadDir("./downloadables")
			if err != nil {
				fmt.Fprintln(conn, "error")
				fmt.Fprintln(conn, "done")
				return
			}	
			targetDir := entries[filenum - 1].Name()
			dirPath := "./downloadables/" + targetDir
			
			files, err := os.ReadDir(dirPath)
			if err != nil {
				fmt.Fprintln(conn, "error")
				fmt.Fprintln(conn, "done")
				return
			}
			var targetFile os.DirEntry
			for _, f := range files {
				if f.Name() != "info.txt" {
					targetFile = f
					break
				}
			}
			fmt.Fprintln(conn, targetFile.Name())

			filedata, err := os.Open(dirPath + "/" + targetFile.Name())
			if err != nil {
				fmt.Fprintln(conn, "error")
				fmt.Fprintln(conn, "done")
				return
			}
			defer filedata.Close()
			info, _ := filedata.Stat()
			fmt.Fprintln(conn, info.Size()) 
			io.Copy(conn, filedata)
		}
	}
}

func client_browse(conn net.Conn){
	entries, err := os.ReadDir("./downloadables")
	if err != nil {
		panic(err)
	}
	if len(entries) == 0 {
		fmt.Println("You have no files to share! Try dragging files into your \"downloadables\" folder.")
	} else {
		for index, entry := range entries{
			//fmt.Printf("%d \n", index+1)
			var name = entry.Name()
			fmt.Fprintf(conn, "%d) %v ", index+1, name)
			var location = "./downloadables/" + name + "/info.txt"
			data, err := os.ReadFile(location)
			if err != nil {
				panic(err)
			}
			var newlineloc = strings.Index(string(data), "\n")
			fmt.Fprintf(conn, "%.*s\n", newlineloc, data)
			newlineloc  += 1
		}
	}
}


func _client_exit(conn net.Conn){
	conn.Close()
	clientsMutex.Lock()
	for i, c := range clients {
		if c == conn {
			clients = append(clients[:i], clients[i+1:]...)
			break
		}
	}
	clientsMutex.Unlock()
}

func listing() int {
	clientsMutex.Lock()
    defer clientsMutex.Unlock()


	fmt.Printf("There are %d connections. These are the current connections:\n", len(clients))
	for i, conn := range clients{
		fmt.Printf("Connection %d: from %v\n", i+1, conn.RemoteAddr())
	}

	return 1
}

func ufile() int {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Type in the file you've updated.")
	scanner.Scan()
	text := scanner.Text()
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "error reading:", err)
	}
	var location = "./downloadables/" + text + "/info.txt"

	data, err := os.ReadFile(location)
	if errors.Is(err, os.ErrNotExist){
		fmt.Println("Invalid file given, check valid files with command 'viewfiles'")
	} else if err != nil{
		panic(err)
	} else{
		s_data := string(data)
		var newlineloc = strings.Index(s_data, "\n")
		int_data := s_data[9:newlineloc]
		ver, err := strconv.Atoi(int_data)
		if err != nil {
			panic(err)
		}
		ver += 1
		err3 := os.WriteFile(location, []byte("version: " + strconv.Itoa(ver) + "\nname: " + text), 0644)
		if err3 != nil {
			panic(err3)
		}
	}
	return 1
}

func main() {
	cmds := make(map[string]execFunc)
	greating(&cmds)
    scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		
		text := scanner.Text()
		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "error reading:", err)
		}
		result := cmd_checker(cmds, text)
		if result != nil{
			result()
		} else {
			fmt.Println("Invalid Command! Try something else.")
		}
		if text == "exit"{
			break
		}	
		fmt.Print("Type your command:")	
	}
}


func greating(cmds *map[string]execFunc) {
	fmt.Println("Welcome to SAL store -serverhost. Version 2")
	fmt.Println("Type help to find out about current commmands and what they do.")
	fmt.Print("Type your command:")

	(*cmds)["help"] = helpmenu
	(*cmds)["exit"] = cexit
	(*cmds)["viewfiles"] = vfiles
	(*cmds)["updatefile"] = ufile
	(*cmds)["start"] = star
	(*cmds)["list"] = listing
}

func cmd_checker(cmds map[string]execFunc , text string) execFunc{
	val, ok := cmds[text]
	if ok {
		return val
	} else {
		return nil
	}
}