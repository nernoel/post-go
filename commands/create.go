package commands

import (
	"bufio"
	"fmt"
	"os"
)

func createNewDatabase() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Enter the name of the database to be created!")
	scanner.Scan()
	databaseName := scanner.Text()

	if err, ok := db.Query("CREATE DATABASE %s", databaseName) {
		fmt.Print(err)
	}


}
