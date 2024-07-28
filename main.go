package main

import (
	"bufio"
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

func GetDatabaseConnStr(c *DatabaseConfig) (connStr string) {
	scanner := bufio.NewScanner(os.Stdin)

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

	connStr = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", c.User, c.Password, c.Host, c.Port, c.Database)
	return
}

func main() {
	// config := &DatabaseConfig{}
	// connStr := GetDatabaseConnStr(config)
	// fmt.Println(connStr) // print for testing
}
