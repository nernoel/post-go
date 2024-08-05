package commands

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"
)

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

/* Global variables */
var scanner = bufio.NewScanner(os.Stdin)

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

/*
 * Function needs to print a list of all databases
 * Possibly with db information in
 * a table styling
 */
func GetListOfDbNames(db *sql.DB) ([]string, error) {
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
		// Add the databases to the database name
		databases = append(databases, dbName)
	}

	// Check for errors scanning rows
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return databases, nil
}

func createNewDatabase(db *sql.DB) {
	/*
	 * Read name for database to be created
	 * Prompt user to enter name and read with bufio
	 */
	fmt.Print("Enter the name of the database to be created: ")
	scanner.Scan()
	databaseName := scanner.Text()

	/*
	 * Execute db query
	 * If error return error message
	 */
	db_query := fmt.Sprintf("CREATE DATABASE %s", databaseName)
	_, err := db.Exec(db_query) // omit result
	if err != nil {
		fmt.Println("Error creating new database!")
		return // return if error executes
	}
}

func deleteDatabase(db *sql.DB) {
	/*
	 * Read name for database to be created
	 * Prompt user to enter name and read with bufio
	 */
	fmt.Print("Enter the name of the database to be deleted: ")
	scanner.Scan()
	databaseName := scanner.Text()
	fmt.Print("Deleting...")
	time.Sleep(4 * time.Second)
	db_query := fmt.Sprintf("DROP DATABASE %s", databaseName)
	_, err := db.Exec(db_query)
	if err != nil {
		fmt.Println("Error deleting daatbase")
		return
	}

}

func readDataabse(db *sql.DB) {
	/* Read the contents for a database
	 *
	 */
}