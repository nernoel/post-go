package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
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

func listAllDatabases(db *sql.DB) ([]string, error) {
	// Selects all databases from user database
	query := "SELECT datname FROM pg_database WHERE datistemplate = false"
	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	// Iterate through the databases Slice to return
	var databases []string
	for rows.Next() {
		var dbName string
		if err := rows.Scan(&dbName); err != nil {
			log.Fatal("Error scanning row")
		}
		databases = append(databases, dbName)
	}

	// Check for errors scanning rows
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return databases, nil
}

func main() {
	isConnected := false
	dbConfig := &DatabaseConfig{}

	if CheckDatabaseConn(dbConfig) {
		isConnected = true
	}
	/*
	 * Continuously run program while db is connected
	 * User is able to enter more simple commands instead of large queries to perform commands
	 * on their postgresql database
	 * NOTE: Consider adding query to print to standard out to show user ??
	 */
	for isConnected {

	}

}
