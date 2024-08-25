package main

import (
	"fmt"
	"postgo/commands" // import file of commands
	"time"

	_ "github.com/lib/pq"
)

func main() {
	// HashMap to map keys to command strings
	commandMap := make(map[rune]string)

	// Adding commands to hashMap
	commandMap['c'] = "CREATE_DATABASE"
	// Define the database config for the user
	dbConfig := &commands.DatabaseConfig{} // set db config to database config from commands
	// Check if the database is connected
	isConnected := commands.CheckDatabaseConn(dbConfig)

	// If the database is successfully connected.. start prompting for commands
	if isConnected == true {
		time.Sleep(5 * time.Second) // Sleep for 5 seconds
		fmt.Print("Welcome back!\n Start entering your commands.\nUse the h key to view all commands")
		for {

		}
	}
}

