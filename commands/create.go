package commands

import (
	"bufio"
	"fmt"
	"os"

	"database/sql"
)

func createNewDatabase(db *sql.DB) {
	/*
	 * Read name for database to be created
	 * Prompt user to enter name and read with bufio
	 */
	scanner := bufio.NewScanner(os.Stdin)
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
