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
	"encoding/json"
)
type execFunc func () int
var theListener net.Listener
var clients []net.Conn
var clientsMutex sync.Mutex
var accountreqsMutex sync.Mutex
var accounts = make(map[string]string)
type accountReq struct {
    username string
    password string
}
var accountreqs []accountReq

func helpmenu() int{
	fmt.Println("All commmands are case sensitive, so be careful when typing them in. \nCurrent commands are the following:")
	fmt.Println("help - prints out text that lists out all commands and then describes each one.")
	fmt.Println("exit - closes the program")
	fmt.Println("viewfiles - lists out the files people will be able to download from you.")
	fmt.Println("updatefile - will ask you for a file name, increments the version value of the file.")
	fmt.Println("start - commences sever to allow connection")
	fmt.Println("list - prints all current connections to the server")

	fmt.Println("acceptreq - see all account creation requests and choose to accept or reject requests")
	fmt.Println("deleteacc - removes an account")
	return 1
}

func cexit() int{
	fmt.Println("clean")
	fmt.Println("exit.")
	if theListener != nil{
		clientsMutex.Lock()
		for _, client := range clients {
			client.Close()
		}
		clientsMutex.Unlock()		
		theListener.Close()
	}
	_storing_accs()
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
	var logged = ""
	scanner := bufio.NewScanner(conn)
	fmt.Fprintln(conn, "hello welcome to this database, login before you doing anything!")

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "error reading:", err)
	}

	for scanner.Scan(){
		
		text := scanner.Text()
		fmt.Printf("Command recivied: %v\n", text)

		if text == "login"{
			client_login(conn, scanner, &logged)
			fmt.Fprintln(conn, "done")
		} else if text == "createacc"{
			client_create(conn, scanner)
			fmt.Fprintln(conn, "done")
		} else if logged == "" {
			fmt.Fprintln(conn, "you aren't logged in! the server is not usable until you log in")
			fmt.Fprintln(conn, "done")
		} else if text == "viewlogin" {
			client_viewlogin(conn, logged)
			fmt.Fprintln(conn, "done")
		} else if text == "logout" {
			client_logout(conn, &logged)
			fmt.Fprintln(conn, "done")
		} else if text == "browse"{
			client_browse(conn)
			fmt.Fprintln(conn, "done")
		} else if text == "download"{
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

func delacc() int {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Please enter the username of the account you want to delete")
	scanner.Scan()
	acctext := scanner.Text()
	for !_logcheck(acctext) {
		fmt.Println("Invalid username, please try again")
		scanner.Scan()
		acctext = scanner.Text()

		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "error reading:", err)
		}
		
	}
	username := acctext
	fmt.Printf("Hello %v, please enter the account's password\n", username)
	scanner.Scan()
	acctext = scanner.Text()
	for _passcheck(acctext, username){
		fmt.Println("Invalid/Incorrect password, please try again")
		scanner.Scan()
		acctext = scanner.Text()

		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "error reading:", err)
		}
		fmt.Printf("result: %v\n", acctext)
	}

	delete(accounts, username)

	return 1
}

func _passcheck(input string, username string) bool {
	if input == "" {
		return false
	}
	return accounts[username] != input
}

func _logcheck(input string) bool {
	if input == "" {
		return false
	}
	for key := range accounts{
		if key == input {
			return true
		}
	}	
	return false
}

func client_login(conn net.Conn, scanner *bufio.Scanner, logged *string) int {

	if *logged != ""{
		fmt.Fprintln(conn, ".")
		return 1
	} 
	fmt.Fprintln(conn, "proc")


	scanner.Scan()
	acctext := scanner.Text()
	for !_logcheck(acctext) {
		fmt.Fprintln(conn, "not ok")
		scanner.Scan()
		acctext = scanner.Text()

		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "error reading:", err)
		}
		
	}
	fmt.Fprintln(conn, "ok")
	username := acctext

	scanner.Scan()
	acctext = scanner.Text()
	for _passcheck(acctext, username){
		fmt.Fprintln(conn, "not ok")
		scanner.Scan()
		acctext = scanner.Text()

		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "error reading:", err)
		}
		fmt.Printf("result: %v\n", acctext)
	}
	fmt.Fprintln(conn, "ok")

	*logged = username
	return 1
}

func client_viewlogin(conn net.Conn, logged string) int {
	fmt.Fprintln(conn, "ok")
	fmt.Fprintln(conn, logged)
	return 1
}

func client_logout(conn net.Conn, logged *string) int {
	fmt.Fprintln(conn, "ok")
	fmt.Fprintf(conn, "%v has been logged out\n", *logged)
	*logged = ""
	
	return 1
}

func _account_making_check(input string) bool {
	if input == "" {
		return true
	}
	_, ok := accounts[input]

	return ok
}

func client_create(conn net.Conn, scanner *bufio.Scanner) int {

	scanner.Scan()
	acctext := scanner.Text()
	for _account_making_check(acctext){
		fmt.Fprintln(conn, "not ok")
		scanner.Scan()
		acctext = scanner.Text()

		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "error reading:", err)
		}
	}
	fmt.Fprintln(conn, "ok")
	username := acctext

	scanner.Scan()
	acctext = scanner.Text()
	password := acctext
	accountreqsMutex.Lock()
	accountreqs = append(accountreqs, accountReq{username: username, password: password})
	accountreqsMutex.Unlock()
	return 1;
}

func areq() int {
	fmt.Println("The following are all requests, type the request number you want to deal with. Type 0 to cancel.")
	for i, request := range accountreqs{
		fmt.Printf("%d) %v : %v\n", i + 1, request.username, request.password)

	}
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	text := scanner.Text()
	num, err2 := strconv.Atoi(text)
	for err2 != nil || num < 0 || num > len(accountreqs) {
        fmt.Println("Invalid input! Proper use: <req_num> or 0. ", err2)

		scanner.Scan()
		text := scanner.Text()
		num, err2 = strconv.Atoi(text)

		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "error reading:", err)
		}
	}
	if num != 0 {
		fmt.Println("type y to accept or r to reject.")
		scanner.Scan()
		ans := scanner.Text()
		for ans != "y" && ans != "n" {
        	fmt.Println("Invalid input! type y to accept or r to reject.")

			scanner.Scan()
			ans = scanner.Text()
			if err := scanner.Err(); err != nil {
				fmt.Fprintln(os.Stderr, "error reading:", err)
			}
		}
		accountreqsMutex.Lock()
		if ans == "y" {
			accounts[accountreqs[num-1].username] = accountreqs[num-1].password
			accountreqs = append(accountreqs[:num-1], accountreqs[num:]...)		
			fmt.Println("account has been accepted.")
		} else {
			accountreqs = append(accountreqs[:num-1], accountreqs[num:]...)		
			fmt.Println("account has been rejected.")
		}
		accountreqsMutex.Unlock()
	}
	return 1
}

func main() {
	cmds := make(map[string]execFunc)
	greating(&cmds, &accounts)
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

func greating(cmds *map[string]execFunc, accounts *map[string]string) {
	fmt.Println("Welcome to SAL store -serverhost. Version 2")
	fmt.Println("Type help to find out about current commmands and what they do.")
	fmt.Print("Type your command:")

	(*cmds)["help"] = helpmenu
	(*cmds)["exit"] = cexit
	(*cmds)["viewfiles"] = vfiles
	(*cmds)["updatefile"] = ufile
	(*cmds)["start"] = star
	(*cmds)["list"] = listing
	(*cmds)["deleteacc"] = delacc
	(*cmds)["acceptreq"] = areq

	_loading_accs(accounts)
}

func cmd_checker(cmds map[string]execFunc , text string) execFunc{
	val, ok := cmds[text]
	if ok {
		return val
	} else {
		return nil
	}
}

func _loading_accs(accounts *map[string]string){
	accinfo, errg := os.ReadFile("./accounts.JSON")
	if errg != nil {
		return
	}
	checkvalid := json.Valid(accinfo)
	if !checkvalid {
		print("account info file is broken")
		return
	}
	json.Unmarshal(accinfo, accounts)
}

func _storing_accs(){
	finalJson, _ := json.MarshalIndent(accounts, "", "\t") 
	errfile := os.WriteFile("./accounts.JSON", finalJson, 0666)
	if errfile != nil {
		panic(errfile)
	}
}
