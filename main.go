package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)


type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

func GetDatabaseConnInfo(c *DatabaseConfig) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Enter the database connection details\n")

	fmt.Print("Enter the Host: ")
	scanner.Scan()
	c.Host = scanner.Text()

	fmt.Print("Enter the Port: ")
	scanner.Scan()
	c.Port = scanner.Text()

	fmt.Print("Enter the User: ")
	scanner.Scan()
	c.User = scanner.Text()

	fmt.Print("Enter the Password: ")
	scanner.Scan()
	c.Password = scanner.Text()

	fmt.Print("Enter the Database name: ")
	scanner.Scan()
	c.Database = scanner.Text()
}

func CheckDatabaseConn(dbConfig *DatabaseConfig) bool {
	// Get the connection info from the user
	GetDatabaseConnInfo(dbConfig)

	// Define the db connection string
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbConfig.Host, dbConfig.Port, dbConfig.User, dbConfig.Password, dbConfig.Database)

	// Attempt to connect to the database
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		fmt.Println("There was an issue connecting to the database... Please try again!")
		return false
	}
	defer db.Close()

	// Verify the connection is alive
	err = db.Ping()
	if err != nil {
		fmt.Println("There was an issue connecting to the database... Please try again!")
		return false
	}

	fmt.Println("Successfully connected to the database!")
	return true
}

func DisplayCommands() {
	
}

func showAllDatabases() {

}


func main() {
	dbConfig := &DatabaseConfig{}
	
	/*
	 * If database returns true then run the main application 
	 */
	if CheckDatabaseConn(dbConfig) == true {
		//
	}



	
}
