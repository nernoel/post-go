package main

import (
	"bufio"
	"fmt"
	"os"
	"postgo/commands" // import file of commands
	"time"

	_ "github.com/lib/pq"
)



func main() {
	dbConfig := &commands.DatabaseConfig{} // set db config to database config from commands
	isConnected := commands.CheckDatabaseConn(dbConfig)
	
	/* Declaring buffered reader for user input */
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Welcome back!\n")
	time.Sleep(5 * time.Second)
	fmt.Print("Press h for any help on the custom commands!")
	fmt.

	/*
	 * Continuously read commands from the user with while loop
	*/
	for {
		//
	} 

}
