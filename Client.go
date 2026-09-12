package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

var accounts = make(map[string]string)
var logged = ""
var conn net.Conn = nil
var connScanner *bufio.Scanner = nil
var connReader *bufio.Reader = nil
type execFunc func () int

func helpmenu() int{
	fmt.Println("All commmands are case sensitive, so be careful when typing them in. \n Current commands are the following:")
	fmt.Println("help - prints out text that lists out all commands and then describes each one.")
	fmt.Println("exit - closes the program")
	fmt.Println("login - will ask for a username and a password, if combination is valid, you be logged in as that user")
	fmt.Println("logout - logs out of account if logged into one")
	fmt.Println("createacc - will ask for a username and a password, will create an account with that information")
	fmt.Println("viewlogin - prints out the user that is currently logged in")
	fmt.Println("deleteacc - removes an account")
	fmt.Println("connect - connects to the server given")
	fmt.Println("disconnect - disconnects from currently connected server")
	fmt.Println("browse - queries connected host for current files")
	fmt.Println("downloadfile - will print all accessible files and then ask which file you'd like to download")

	return 1
}

func cexit() int{
	logged = ""
	fmt.Println("clean")
	fmt.Println("exit.")
	_storing_accs()
	

	return 1
}

func connect() int{
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Please enter what you'd like to connect to")
	scanner.Scan()
	acctext := scanner.Text()
	var dialerr error
	conn, dialerr = net.Dial("tcp", acctext)

	if dialerr != nil {
		fmt.Println("Failed to connect (you likely gave an invalid connection):", dialerr)
        return 1
	}
	connReader = bufio.NewReader(conn)
	connScanner = bufio.NewScanner(connReader) 
    connScanner.Scan()
	fmt.Print("Server says: ", connScanner.Text())

	return 1
}

func browse() int {

	if conn == nil {
		fmt.Println("Failed to browse (you arent connected to anything)")
		return 1
	}

	fmt.Fprintln(conn, "browse")	
	
	for connScanner.Scan(){

		
		text := connScanner.Text()

		if text == "done" {
            break
        }

		fmt.Printf("Command recivied: %v\n", text)

		
	}
	
	if err := connScanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "error reading:", err)
	}

	return 1
}

func downfile() int{
	fmt.Fprintln(conn, "browse")	
	filecounter := 0
	for connScanner.Scan(){
		
		text := connScanner.Text()

		if text == "done" {
            break
        }

		fmt.Printf("Command recivied: %v\n", text)
		filecounter += 1
		
	}
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Please select the file you'd like to download by typing their associated number. (Type 0 to cancel)")	
	scanner.Scan()
	fileTBD := scanner.Text()
	for _filecheck(fileTBD, filecounter){
		fmt.Println("Input a vaild file number:")
		scanner.Scan()
		fileTBD = scanner.Text()

		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "error reading:", err)
		}

	}


	if fileTBD != "0" {
		fmt.Fprintln(conn, "download")
		fmt.Fprintln(conn, fileTBD)

		//recieve the file that will now be coming
		filename, _ := connReader.ReadString('\n')
		filename = strings.TrimSpace(filename)

		sizeStr, _ := connReader.ReadString('\n')
		filesize, _ := strconv.ParseInt(strings.TrimSpace(sizeStr), 10, 64)

		newfile, err := os.Create("./" + filename)
		if err != nil {
			fmt.Println("Failed to create file: ", err)
			return 1
		}
		defer newfile.Close()

		blah, err := io.Copy(newfile, io.LimitReader(connReader, filesize))
		if err != nil{
			fmt.Println("Error during download:", err)
			fmt.Println("\npt2:", blah)
			return 1
		}

		fmt.Println("Download complete:", filename)

	}

	return 1
}

func _filecheck(input string, cap int) bool {

	num, err := strconv.Atoi(input)
	if err != nil {
		fmt.Println("Please input a number!")
		return true
	}
	if num == 0 {
		fmt.Println("Canceling Download.")
		return false
	}
	if num > cap || num < 0 {
		fmt.Println("There's no file associated with this number. Try something else")
		return true
	}

	return false
}

func login() int {

	if logged != ""{
		fmt.Println("An account is already logged in!")
		return 1
	}

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Please enter a username")
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
	fmt.Printf("Hello %v, please enter your password\n", username)
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

	fmt.Println("You are now logged in!")
	logged = username
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

func _account_making_check(input string) bool {
	if input == "" {
		return true
	}
	_, ok := accounts[input]

	return ok
}

func cac() int {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Please enter your new username")
	scanner.Scan()
	acctext := scanner.Text()
	for _account_making_check(acctext){
		fmt.Println("Invalid/Already in use username, please try again")
		scanner.Scan()
		acctext = scanner.Text()

		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "error reading:", err)
		}
	}
	username := acctext
	fmt.Printf("Hello %v, please enter what you'd like your password to be\n", username)
	scanner.Scan()
	acctext = scanner.Text()
	for acctext == ""{
		fmt.Println("Invalid password, please try again")
		scanner.Scan()
		acctext = scanner.Text()

		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "error reading:", err)
		}
	}
	password := acctext
	accounts[username] = password
	fmt.Println("Account created!")

	return 1;
}

func vlogin() int {
	if logged != ""{
		fmt.Printf("The user currently logged in is: %v\n", logged)
	} else {
		fmt.Println("No user is currently logged in.")
	}
	return 1
}

func logo() int {
	if logged != ""{
		fmt.Printf("%v has been logged out\n", logged)
		logged = ""
	} else {
		fmt.Println("No user is currently logged in.")
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

func main() {
	cmds := make(map[string]execFunc)
	
	greating(&cmds, &accounts)
    scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		text := scanner.Text()
		result := cmd_checker(cmds, text)
		if result != nil{
			result()
		} else {
			fmt.Println("Invalid Command! Try something else.")
		}
	
		if text == "exit"{
			break
		}

		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "error reading:", err)
		}
	}
}


func greating(cmds *map[string]execFunc, accounts *map[string]string) {
	fmt.Println("Welcome to SAL store -client. Version 1")
	fmt.Println("Type help to find out about current commmands and what they do.")

	(*cmds)["help"] = helpmenu
	(*cmds)["exit"] = cexit
	(*cmds)["login"] = login
	(*cmds)["createacc"] = cac
	(*cmds)["viewlogin"] = vlogin
	(*cmds)["logout"] = logo
	(*cmds)["deleteacc"] = delacc
	(*cmds)["connect"] = connect
	(*cmds)["browse"] = browse
	(*cmds)["downloadfile"] = downfile
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