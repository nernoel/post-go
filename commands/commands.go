package commands

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	_ "github.com/lib/pq"
)

const (
	appName = "postgo"
	version = "1.0.0"
)

var commonTypes = []string{
	"SERIAL PRIMARY KEY",
	"BIGSERIAL PRIMARY KEY",
	"INTEGER",
	"BIGINT",
	"NUMERIC(12,2)",
	"TEXT",
	"VARCHAR(255)",
	"BOOLEAN",
	"DATE",
	"TIMESTAMP",
	"TIMESTAMPTZ",
	"UUID",
	"JSONB",
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	SSLMode  string
}

type app struct {
	in       *bufio.Scanner
	out      io.Writer
	db       *sql.DB
	config   DatabaseConfig
	commands map[string]command
	aliases  map[string]string
}

type command struct {
	Name        string
	Description string
	Aliases     []string
	NeedsDB     bool
	Run         func(context.Context, *app) error
}

func Run(args []string) error {
	fs := flag.NewFlagSet(appName, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	showHelp := fs.Bool("help", false, "show help")
	showVersion := fs.Bool("version", false, "show version")
	if err := fs.Parse(args); err != nil {
		return err
	}

	a := newApp(os.Stdin, os.Stdout)
	if *showVersion {
		fmt.Fprintf(a.out, "%s %s\n", appName, version)
		return nil
	}
	if *showHelp {
		a.printIntro()
		a.printHelp()
		return nil
	}

	return a.loop(context.Background())
}

func newApp(in io.Reader, out io.Writer) *app {
	a := &app{
		in:      bufio.NewScanner(in),
		out:     out,
		aliases: map[string]string{},
	}
	a.commands = a.buildCommands()
	for name, cmd := range a.commands {
		a.aliases[name] = name
		for _, alias := range cmd.Aliases {
			a.aliases[alias] = name
		}
	}
	return a
}

func (a *app) loop(ctx context.Context) error {
	a.printIntro()
	if yes(a.askDefault("Connect to a Postgres database now?", "y")) {
		if err := a.connect(ctx); err != nil {
			fmt.Fprintf(a.out, "Connection failed: %v\n", err)
		}
	}

	for {
		fmt.Fprint(a.out, "\npostgo> ")
		if !a.in.Scan() {
			break
		}
		input := strings.TrimSpace(a.in.Text())
		if input == "" {
			continue
		}
		if err := a.runCommand(ctx, input); err != nil {
			if errors.Is(err, errExit) {
				break
			}
			fmt.Fprintf(a.out, "Error: %v\n", err)
		}
	}
	if err := a.in.Err(); err != nil {
		return err
	}
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}

func (a *app) runCommand(ctx context.Context, input string) error {
	name, ok := a.aliases[strings.ToLower(input)]
	if !ok {
		fmt.Fprintf(a.out, "Unknown command %q. Type help to see what Postgo can do.\n", input)
		return nil
	}
	cmd := a.commands[name]
	if cmd.NeedsDB && a.db == nil {
		fmt.Fprintln(a.out, "This command needs a database connection first.")
		return a.connect(ctx)
	}
	return cmd.Run(ctx, a)
}

func (a *app) printIntro() {
	fmt.Fprintf(a.out, "%s %s - guided SQL from the terminal\n", appName, version)
	fmt.Fprintln(a.out, "Type help for commands, connect to change databases, or quit to exit.")
}

func (a *app) printHelp() {
	names := make([]string, 0, len(a.commands))
	for name := range a.commands {
		names = append(names, name)
	}
	sort.Strings(names)

	fmt.Fprintln(a.out, "\nAvailable commands:")
	for _, name := range names {
		cmd := a.commands[name]
		aliases := ""
		if len(cmd.Aliases) > 0 {
			aliases = " (" + strings.Join(cmd.Aliases, ", ") + ")"
		}
		fmt.Fprintf(a.out, "  %-18s %s%s\n", cmd.Name+aliases, cmd.Description, needsDB(cmd.NeedsDB))
	}
}

func needsDB(needs bool) string {
	if needs {
		return " [db]"
	}
	return ""
}

func (a *app) buildCommands() map[string]command {
	return map[string]command{
		"help":              {Name: "help", Description: "show this command list", Aliases: []string{"h", "?"}, Run: func(_ context.Context, a *app) error { a.printHelp(); return nil }},
		"connect":           {Name: "connect", Description: "connect or reconnect to Postgres", Aliases: []string{"conn", "login"}, Run: func(ctx context.Context, a *app) error { return a.connect(ctx) }},
		"status":            {Name: "status", Description: "show current connection info", Aliases: []string{"whoami"}, Run: status},
		"list-db":           {Name: "list-db", Description: "list databases", Aliases: []string{"databases", "dbs"}, NeedsDB: true, Run: listDatabases},
		"create-db":         {Name: "create-db", Description: "create a database", Aliases: []string{"createdb"}, NeedsDB: true, Run: createDatabase},
		"drop-db":           {Name: "drop-db", Description: "drop a database", Aliases: []string{"deletedb"}, NeedsDB: true, Run: dropDatabase},
		"list-schemas":      {Name: "list-schemas", Description: "list schemas", Aliases: []string{"schemas"}, NeedsDB: true, Run: listSchemas},
		"create-schema":     {Name: "create-schema", Description: "create a schema", Aliases: []string{"schema-create"}, NeedsDB: true, Run: createSchema},
		"drop-schema":       {Name: "drop-schema", Description: "drop a schema", Aliases: []string{"schema-drop"}, NeedsDB: true, Run: dropSchema},
		"list-tables":       {Name: "list-tables", Description: "list tables", Aliases: []string{"tables", "lt"}, NeedsDB: true, Run: listTables},
		"describe-table":    {Name: "describe-table", Description: "show table columns", Aliases: []string{"describe", "desc"}, NeedsDB: true, Run: describeTable},
		"show-columns":      {Name: "show-columns", Description: "show column names for a table", Aliases: []string{"columns"}, NeedsDB: true, Run: showColumns},
		"create-table":      {Name: "create-table", Description: "build CREATE TABLE step by step", Aliases: []string{"ct"}, NeedsDB: true, Run: createTable},
		"drop-table":        {Name: "drop-table", Description: "drop a table", Aliases: []string{"dt"}, NeedsDB: true, Run: dropTable},
		"truncate-table":    {Name: "truncate-table", Description: "empty a table", Aliases: []string{"truncate"}, NeedsDB: true, Run: truncateTable},
		"rename-table":      {Name: "rename-table", Description: "rename a table", Aliases: []string{"rt"}, NeedsDB: true, Run: renameTable},
		"add-column":        {Name: "add-column", Description: "add a column", Aliases: []string{"ac"}, NeedsDB: true, Run: addColumn},
		"drop-column":       {Name: "drop-column", Description: "drop a column", Aliases: []string{"dc"}, NeedsDB: true, Run: dropColumn},
		"rename-column":     {Name: "rename-column", Description: "rename a column", Aliases: []string{"rc"}, NeedsDB: true, Run: renameColumn},
		"alter-column-type": {Name: "alter-column-type", Description: "change a column type", Aliases: []string{"type-column"}, NeedsDB: true, Run: alterColumnType},
		"set-not-null":      {Name: "set-not-null", Description: "require a column value", Aliases: []string{"not-null"}, NeedsDB: true, Run: setNotNull},
		"drop-not-null":     {Name: "drop-not-null", Description: "allow NULL in a column", Aliases: []string{"allow-null"}, NeedsDB: true, Run: dropNotNull},
		"add-primary-key":   {Name: "add-primary-key", Description: "add a primary key constraint", Aliases: []string{"pk"}, NeedsDB: true, Run: addPrimaryKey},
		"add-foreign-key":   {Name: "add-foreign-key", Description: "add a foreign key constraint", Aliases: []string{"fk"}, NeedsDB: true, Run: addForeignKey},
		"drop-constraint":   {Name: "drop-constraint", Description: "drop a table constraint", Aliases: []string{"dcst"}, NeedsDB: true, Run: dropConstraint},
		"list-constraints":  {Name: "list-constraints", Description: "list table constraints", Aliases: []string{"constraints"}, NeedsDB: true, Run: listConstraints},
		"create-index":      {Name: "create-index", Description: "create an index", Aliases: []string{"ci"}, NeedsDB: true, Run: createIndex},
		"drop-index":        {Name: "drop-index", Description: "drop an index", Aliases: []string{"di"}, NeedsDB: true, Run: dropIndex},
		"list-indexes":      {Name: "list-indexes", Description: "list indexes", Aliases: []string{"indexes"}, NeedsDB: true, Run: listIndexes},
		"insert-row":        {Name: "insert-row", Description: "insert one row", Aliases: []string{"insert"}, NeedsDB: true, Run: insertRow},
		"select-rows":       {Name: "select-rows", Description: "read rows with filters", Aliases: []string{"select", "find"}, NeedsDB: true, Run: selectRows},
		"update-rows":       {Name: "update-rows", Description: "update rows with a WHERE clause", Aliases: []string{"update"}, NeedsDB: true, Run: updateRows},
		"delete-rows":       {Name: "delete-rows", Description: "delete rows with a WHERE clause", Aliases: []string{"delete"}, NeedsDB: true, Run: deleteRows},
		"count-rows":        {Name: "count-rows", Description: "count rows in a table", Aliases: []string{"count"}, NeedsDB: true, Run: countRows},
		"create-view":       {Name: "create-view", Description: "create a view from a SELECT", Aliases: []string{"cv"}, NeedsDB: true, Run: createView},
		"drop-view":         {Name: "drop-view", Description: "drop a view", Aliases: []string{"dv"}, NeedsDB: true, Run: dropView},
		"list-views":        {Name: "list-views", Description: "list views", Aliases: []string{"views"}, NeedsDB: true, Run: listViews},
		"list-roles":        {Name: "list-roles", Description: "list database roles", Aliases: []string{"roles"}, NeedsDB: true, Run: listRoles},
		"create-role":       {Name: "create-role", Description: "create a database role", Aliases: []string{"role-create"}, NeedsDB: true, Run: createRole},
		"grant-table":       {Name: "grant-table", Description: "grant table privileges", Aliases: []string{"grant"}, NeedsDB: true, Run: grantTable},
		"revoke-table":      {Name: "revoke-table", Description: "revoke table privileges", Aliases: []string{"revoke"}, NeedsDB: true, Run: revokeTable},
		"analyze-table":     {Name: "analyze-table", Description: "refresh planner statistics", Aliases: []string{"analyze"}, NeedsDB: true, Run: analyzeTable},
		"vacuum-table":      {Name: "vacuum-table", Description: "reclaim table storage", Aliases: []string{"vacuum"}, NeedsDB: true, Run: vacuumTable},
		"backup-table-as":   {Name: "backup-table-as", Description: "copy a table into a new table", Aliases: []string{"backup-table"}, NeedsDB: true, Run: backupTableAs},
		"explain-query":     {Name: "explain-query", Description: "run EXPLAIN for a SELECT", Aliases: []string{"explain"}, NeedsDB: true, Run: explainQuery},
		"raw-sql":           {Name: "raw-sql", Description: "run SQL manually when needed", Aliases: []string{"sql"}, NeedsDB: true, Run: rawSQL},
		"quit":              {Name: "quit", Description: "exit Postgo", Aliases: []string{"exit", "q"}, Run: func(context.Context, *app) error { return errExit }},
	}
}

var errExit = errors.New("exit")

func (a *app) connect(ctx context.Context) error {
	cfg := DatabaseConfig{
		Host:     firstNonEmpty(os.Getenv("PGHOST"), "localhost"),
		Port:     firstNonEmpty(os.Getenv("PGPORT"), "5432"),
		User:     firstNonEmpty(os.Getenv("PGUSER"), "postgres"),
		Password: os.Getenv("PGPASSWORD"),
		Database: firstNonEmpty(os.Getenv("PGDATABASE"), "postgres"),
		SSLMode:  firstNonEmpty(os.Getenv("PGSSLMODE"), "disable"),
	}

	fmt.Fprintln(a.out, "\nEnter database connection details. Press Enter to keep the default.")
	cfg.Host = a.askDefault("Host", cfg.Host)
	cfg.Port = a.askDefault("Port", cfg.Port)
	cfg.User = a.askDefault("User", cfg.User)
	cfg.Password = a.askDefault("Password", cfg.Password)
	cfg.Database = a.askDefault("Database", cfg.Database)
	cfg.SSLMode = a.askDefault("SSL mode", cfg.SSLMode)

	db, err := sql.Open("postgres", cfg.connString())
	if err != nil {
		return err
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return err
	}
	if a.db != nil {
		a.db.Close()
	}
	a.db = db
	a.config = cfg
	fmt.Fprintf(a.out, "Connected to %s on %s:%s as %s.\n", cfg.Database, cfg.Host, cfg.Port, cfg.User)
	return nil
}

func (c DatabaseConfig) connString() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode)
}

func status(_ context.Context, a *app) error {
	if a.db == nil {
		fmt.Fprintln(a.out, "Not connected.")
		return nil
	}
	fmt.Fprintf(a.out, "Connected to %s on %s:%s as %s.\n", a.config.Database, a.config.Host, a.config.Port, a.config.User)
	return nil
}

func listDatabases(ctx context.Context, a *app) error {
	return printRows(ctx, a, "SELECT datname AS database FROM pg_database WHERE datistemplate = false ORDER BY datname")
}

func createDatabase(ctx context.Context, a *app) error {
	name := a.askIdentifier("New database name")
	return execSQL(ctx, a, "CREATE DATABASE "+quoteIdent(name))
}

func dropDatabase(ctx context.Context, a *app) error {
	name := a.askIdentifier("Database name to drop")
	if strings.EqualFold(name, a.config.Database) {
		return errors.New("connect to a different database before dropping the current one")
	}
	sqlText := "DROP DATABASE " + quoteIdent(name)
	return execSQL(ctx, a, sqlText)
}

func listSchemas(ctx context.Context, a *app) error {
	return printRows(ctx, a, "SELECT schema_name FROM information_schema.schemata ORDER BY schema_name")
}

func createSchema(ctx context.Context, a *app) error {
	name := a.askIdentifier("New schema name")
	return execSQL(ctx, a, "CREATE SCHEMA "+quoteIdent(name))
}

func dropSchema(ctx context.Context, a *app) error {
	name := a.askIdentifier("Schema name to drop")
	cascade := yes(a.askDefault("Cascade and drop contained objects?", "n"))
	sqlText := "DROP SCHEMA " + quoteIdent(name)
	if cascade {
		sqlText += " CASCADE"
	}
	return execSQL(ctx, a, sqlText)
}

func listTables(ctx context.Context, a *app) error {
	return printRows(ctx, a, `SELECT table_schema, table_name
FROM information_schema.tables
WHERE table_type = 'BASE TABLE' AND table_schema NOT IN ('pg_catalog', 'information_schema')
ORDER BY table_schema, table_name`)
}

func describeTable(ctx context.Context, a *app) error {
	table := a.askTableName()
	return printRows(ctx, a, `SELECT column_name, data_type, is_nullable, column_default
FROM information_schema.columns
WHERE table_schema = $1 AND table_name = $2
ORDER BY ordinal_position`, table.Schema, table.Name)
}

func showColumns(ctx context.Context, a *app) error {
	table := a.askTableName()
	return printRows(ctx, a, `SELECT column_name
FROM information_schema.columns
WHERE table_schema = $1 AND table_name = $2
ORDER BY ordinal_position`, table.Schema, table.Name)
}

func createTable(ctx context.Context, a *app) error {
	table := a.askTableName()
	var columns []string
	for {
		name := a.askIdentifier("Column name")
		colType := a.askType()
		nullable := yes(a.askDefault("Allow NULL values?", "y"))
		unique := yes(a.askDefault("Unique column?", "n"))
		defaultValue := strings.TrimSpace(a.askDefault("Default value SQL, blank for none", ""))

		col := quoteIdent(name) + " " + colType
		if !nullable {
			col += " NOT NULL"
		}
		if unique {
			col += " UNIQUE"
		}
		if defaultValue != "" {
			col += " DEFAULT " + defaultValue
		}
		columns = append(columns, col)

		if !yes(a.askDefault("Add another column?", "y")) {
			break
		}
	}
	sqlText := fmt.Sprintf("CREATE TABLE %s (\n  %s\n)", table.SQL(), strings.Join(columns, ",\n  "))
	return execSQL(ctx, a, sqlText)
}

func dropTable(ctx context.Context, a *app) error {
	table := a.askTableName()
	cascade := yes(a.askDefault("Cascade and drop dependent objects?", "n"))
	sqlText := "DROP TABLE " + table.SQL()
	if cascade {
		sqlText += " CASCADE"
	}
	return execSQL(ctx, a, sqlText)
}

func truncateTable(ctx context.Context, a *app) error {
	table := a.askTableName()
	restart := yes(a.askDefault("Restart identity columns?", "n"))
	sqlText := "TRUNCATE TABLE " + table.SQL()
	if restart {
		sqlText += " RESTART IDENTITY"
	}
	return execSQL(ctx, a, sqlText)
}

func renameTable(ctx context.Context, a *app) error {
	table := a.askTableName()
	newName := a.askIdentifier("New table name")
	sqlText := fmt.Sprintf("ALTER TABLE %s RENAME TO %s", table.SQL(), quoteIdent(newName))
	return execSQL(ctx, a, sqlText)
}

func addColumn(ctx context.Context, a *app) error {
	table := a.askTableName()
	name := a.askIdentifier("New column name")
	colType := a.askType()
	nullable := yes(a.askDefault("Allow NULL values?", "y"))
	sqlText := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table.SQL(), quoteIdent(name), colType)
	if !nullable {
		sqlText += " NOT NULL"
	}
	return execSQL(ctx, a, sqlText)
}

func dropColumn(ctx context.Context, a *app) error {
	table := a.askTableName()
	name := a.askIdentifier("Column name to drop")
	cascade := yes(a.askDefault("Cascade and drop dependent objects?", "n"))
	sqlText := fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", table.SQL(), quoteIdent(name))
	if cascade {
		sqlText += " CASCADE"
	}
	return execSQL(ctx, a, sqlText)
}

func renameColumn(ctx context.Context, a *app) error {
	table := a.askTableName()
	oldName := a.askIdentifier("Current column name")
	newName := a.askIdentifier("New column name")
	sqlText := fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s", table.SQL(), quoteIdent(oldName), quoteIdent(newName))
	return execSQL(ctx, a, sqlText)
}

func alterColumnType(ctx context.Context, a *app) error {
	table := a.askTableName()
	column := a.askIdentifier("Column name")
	colType := a.askType()
	sqlText := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s", table.SQL(), quoteIdent(column), colType)
	return execSQL(ctx, a, sqlText)
}

func setNotNull(ctx context.Context, a *app) error {
	table := a.askTableName()
	column := a.askIdentifier("Column name")
	sqlText := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL", table.SQL(), quoteIdent(column))
	return execSQL(ctx, a, sqlText)
}

func dropNotNull(ctx context.Context, a *app) error {
	table := a.askTableName()
	column := a.askIdentifier("Column name")
	sqlText := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL", table.SQL(), quoteIdent(column))
	return execSQL(ctx, a, sqlText)
}

func addPrimaryKey(ctx context.Context, a *app) error {
	table := a.askTableName()
	constraint := a.askIdentifier("Constraint name")
	columns := a.askIdentifierList("Primary key columns, comma separated")
	sqlText := fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s PRIMARY KEY (%s)", table.SQL(), quoteIdent(constraint), quoteIdentifiers(columns))
	return execSQL(ctx, a, sqlText)
}

func addForeignKey(ctx context.Context, a *app) error {
	table := a.askTableName()
	constraint := a.askIdentifier("Constraint name")
	columns := a.askIdentifierList("Local columns, comma separated")
	referencedTable := a.askTableNameWithPrompt("Referenced table")
	referencedColumns := a.askIdentifierList("Referenced columns, comma separated")
	onDelete := a.askConstraintAction("ON DELETE action")
	onUpdate := a.askConstraintAction("ON UPDATE action")

	sqlText := fmt.Sprintf(
		"ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s)",
		table.SQL(),
		quoteIdent(constraint),
		quoteIdentifiers(columns),
		referencedTable.SQL(),
		quoteIdentifiers(referencedColumns),
	)
	if onDelete != "" {
		sqlText += " ON DELETE " + onDelete
	}
	if onUpdate != "" {
		sqlText += " ON UPDATE " + onUpdate
	}
	return execSQL(ctx, a, sqlText)
}

func dropConstraint(ctx context.Context, a *app) error {
	table := a.askTableName()
	constraint := a.askIdentifier("Constraint name")
	cascade := yes(a.askDefault("Cascade and drop dependent objects?", "n"))
	sqlText := fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s", table.SQL(), quoteIdent(constraint))
	if cascade {
		sqlText += " CASCADE"
	}
	return execSQL(ctx, a, sqlText)
}

func listConstraints(ctx context.Context, a *app) error {
	return printRows(ctx, a, `SELECT tc.table_schema, tc.table_name, tc.constraint_name, tc.constraint_type
FROM information_schema.table_constraints tc
WHERE tc.table_schema NOT IN ('pg_catalog', 'information_schema')
ORDER BY tc.table_schema, tc.table_name, tc.constraint_name`)
}

func createIndex(ctx context.Context, a *app) error {
	table := a.askTableName()
	index := a.askTableNameWithPrompt("Index name")
	columns := a.askIdentifierList("Column names, comma separated")
	unique := yes(a.askDefault("Unique index?", "n"))
	sqlText := "CREATE "
	if unique {
		sqlText += "UNIQUE "
	}
	sqlText += fmt.Sprintf("INDEX %s ON %s (%s)", index.SQL(), table.SQL(), quoteIdentifiers(columns))
	return execSQL(ctx, a, sqlText)
}

func dropIndex(ctx context.Context, a *app) error {
	index := a.askTableNameWithPrompt("Index name")
	return execSQL(ctx, a, "DROP INDEX "+index.SQL())
}

func listIndexes(ctx context.Context, a *app) error {
	return printRows(ctx, a, `SELECT schemaname, tablename, indexname, indexdef
FROM pg_indexes
WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
ORDER BY schemaname, tablename, indexname`)
}

func insertRow(ctx context.Context, a *app) error {
	table := a.askTableName()
	columns := a.askIdentifierList("Column names, comma separated")
	values := make([]string, 0, len(columns))
	args := make([]any, 0, len(columns))
	for _, col := range columns {
		values = append(values, "$"+strconv.Itoa(len(values)+1))
		args = append(args, a.askDefault("Value for "+col, ""))
	}
	sqlText := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table.SQL(), quoteIdentifiers(columns), strings.Join(values, ", "))
	return execSQL(ctx, a, sqlText, args...)
}

func selectRows(ctx context.Context, a *app) error {
	table := a.askTableName()
	columns := strings.TrimSpace(a.askDefault("Columns to return, comma separated or *", "*"))
	if columns != "*" {
		columnNames, err := parseIdentifierList(columns)
		if err != nil {
			return err
		}
		columns = quoteIdentifiers(columnNames)
	}
	where := strings.TrimSpace(a.askDefault("WHERE clause, blank for none", ""))
	limit := strings.TrimSpace(a.askDefault("Limit", "25"))
	sqlText := fmt.Sprintf("SELECT %s FROM %s", columns, table.SQL())
	if where != "" {
		sqlText += " WHERE " + where
	}
	if limit != "" {
		n, err := strconv.Atoi(limit)
		if err != nil || n < 1 {
			return errors.New("limit must be a positive number")
		}
		sqlText += " LIMIT " + strconv.Itoa(n)
	}
	fmt.Fprintf(a.out, "\nSQL:\n%s\n\n", sqlText)
	return printRows(ctx, a, sqlText)
}

func updateRows(ctx context.Context, a *app) error {
	table := a.askTableName()
	columns := a.askIdentifierList("Columns to update, comma separated")
	assignments := make([]string, 0, len(columns))
	args := make([]any, 0, len(columns))
	for _, col := range columns {
		args = append(args, a.askDefault("New value for "+col, ""))
		assignments = append(assignments, fmt.Sprintf("%s = $%d", quoteIdent(col), len(args)))
	}
	where := strings.TrimSpace(a.askRequired("WHERE clause"))
	sqlText := fmt.Sprintf("UPDATE %s SET %s WHERE %s", table.SQL(), strings.Join(assignments, ", "), where)
	return execSQL(ctx, a, sqlText, args...)
}

func deleteRows(ctx context.Context, a *app) error {
	table := a.askTableName()
	where := strings.TrimSpace(a.askRequired("WHERE clause"))
	sqlText := fmt.Sprintf("DELETE FROM %s WHERE %s", table.SQL(), where)
	return execSQL(ctx, a, sqlText)
}

func countRows(ctx context.Context, a *app) error {
	table := a.askTableName()
	where := strings.TrimSpace(a.askDefault("WHERE clause, blank for none", ""))
	sqlText := fmt.Sprintf("SELECT COUNT(*) AS rows FROM %s", table.SQL())
	if where != "" {
		sqlText += " WHERE " + where
	}
	return printRows(ctx, a, sqlText)
}

func createView(ctx context.Context, a *app) error {
	name := a.askTableName()
	selectSQL := strings.TrimSpace(a.askRequired("SELECT query for the view"))
	if !strings.HasPrefix(strings.ToLower(selectSQL), "select") {
		return errors.New("view query must start with SELECT")
	}
	sqlText := fmt.Sprintf("CREATE VIEW %s AS %s", name.SQL(), selectSQL)
	return execSQL(ctx, a, sqlText)
}

func dropView(ctx context.Context, a *app) error {
	name := a.askTableName()
	return execSQL(ctx, a, "DROP VIEW "+name.SQL())
}

func listViews(ctx context.Context, a *app) error {
	return printRows(ctx, a, `SELECT table_schema, table_name
FROM information_schema.views
WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
ORDER BY table_schema, table_name`)
}

func listRoles(ctx context.Context, a *app) error {
	return printRows(ctx, a, `SELECT rolname, rolcanlogin, rolsuper, rolcreatedb, rolcreaterole
FROM pg_roles
ORDER BY rolname`)
}

func createRole(ctx context.Context, a *app) error {
	role := a.askIdentifier("Role name")
	login := yes(a.askDefault("Allow login?", "y"))
	password := strings.TrimSpace(a.askDefault("Password, blank for none", ""))
	createDB := yes(a.askDefault("Allow creating databases?", "n"))
	createRole := yes(a.askDefault("Allow creating roles?", "n"))

	parts := []string{"CREATE ROLE", quoteIdent(role)}
	if login {
		parts = append(parts, "LOGIN")
	} else {
		parts = append(parts, "NOLOGIN")
	}
	if password != "" {
		parts = append(parts, "PASSWORD", quoteLiteral(password))
	}
	if createDB {
		parts = append(parts, "CREATEDB")
	}
	if createRole {
		parts = append(parts, "CREATEROLE")
	}
	return execSQL(ctx, a, strings.Join(parts, " "))
}

func grantTable(ctx context.Context, a *app) error {
	table := a.askTableName()
	role := a.askIdentifier("Role name")
	privileges := a.askPrivileges()
	sqlText := fmt.Sprintf("GRANT %s ON TABLE %s TO %s", strings.Join(privileges, ", "), table.SQL(), quoteIdent(role))
	return execSQL(ctx, a, sqlText)
}

func revokeTable(ctx context.Context, a *app) error {
	table := a.askTableName()
	role := a.askIdentifier("Role name")
	privileges := a.askPrivileges()
	sqlText := fmt.Sprintf("REVOKE %s ON TABLE %s FROM %s", strings.Join(privileges, ", "), table.SQL(), quoteIdent(role))
	return execSQL(ctx, a, sqlText)
}

func analyzeTable(ctx context.Context, a *app) error {
	table := a.askTableName()
	return execSQL(ctx, a, "ANALYZE "+table.SQL())
}

func vacuumTable(ctx context.Context, a *app) error {
	table := a.askTableName()
	full := yes(a.askDefault("Run FULL vacuum? This can lock the table.", "n"))
	analyze := yes(a.askDefault("Analyze after vacuum?", "y"))
	sqlText := "VACUUM"
	if full {
		sqlText += " FULL"
	}
	if analyze {
		sqlText += " ANALYZE"
	}
	sqlText += " " + table.SQL()
	return execSQL(ctx, a, sqlText)
}

func backupTableAs(ctx context.Context, a *app) error {
	source := a.askTableName()
	destination := a.askTableNameWithPrompt("Backup table name")
	sqlText := fmt.Sprintf("CREATE TABLE %s AS TABLE %s", destination.SQL(), source.SQL())
	return execSQL(ctx, a, sqlText)
}

func explainQuery(ctx context.Context, a *app) error {
	query := strings.TrimSpace(a.askRequired("SELECT query to explain"))
	if !strings.HasPrefix(strings.ToLower(query), "select") {
		return errors.New("EXPLAIN helper only accepts SELECT queries")
	}
	return printRows(ctx, a, "EXPLAIN "+query)
}

func rawSQL(ctx context.Context, a *app) error {
	sqlText := strings.TrimSpace(a.askRequired("SQL"))
	if looksLikeQuery(sqlText) {
		return printRows(ctx, a, sqlText)
	}
	return execSQL(ctx, a, sqlText)
}

func execSQL(ctx context.Context, a *app, sqlText string, args ...any) error {
	fmt.Fprintf(a.out, "\nSQL:\n%s\n\n", sqlText)
	if !yes(a.askDefault("Run this SQL?", "y")) {
		fmt.Fprintln(a.out, "Skipped.")
		return nil
	}
	result, err := a.db.ExecContext(ctx, sqlText, args...)
	if err != nil {
		return err
	}
	if rows, err := result.RowsAffected(); err == nil {
		fmt.Fprintf(a.out, "Done. Rows affected: %d\n", rows)
	} else {
		fmt.Fprintln(a.out, "Done.")
	}
	return nil
}

func printRows(ctx context.Context, a *app, query string, args ...any) error {
	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return err
	}
	values := make([]sql.NullString, len(cols))
	dest := make([]any, len(cols))
	widths := make([]int, len(cols))
	for i, col := range cols {
		dest[i] = &values[i]
		widths[i] = len(col)
	}

	var data [][]string
	for rows.Next() {
		if err := rows.Scan(dest...); err != nil {
			return err
		}
		line := make([]string, len(cols))
		for i, value := range values {
			if value.Valid {
				line[i] = value.String
			} else {
				line[i] = "NULL"
			}
			if len(line[i]) > widths[i] {
				widths[i] = len(line[i])
			}
		}
		data = append(data, line)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	printTable(a.out, cols, widths, data)
	return nil
}

func printTable(out io.Writer, cols []string, widths []int, data [][]string) {
	for i, col := range cols {
		fmt.Fprintf(out, "%-*s", widths[i], col)
		if i < len(cols)-1 {
			fmt.Fprint(out, " | ")
		}
	}
	fmt.Fprintln(out)
	for i, width := range widths {
		fmt.Fprint(out, strings.Repeat("-", width))
		if i < len(widths)-1 {
			fmt.Fprint(out, "-+-")
		}
	}
	fmt.Fprintln(out)
	for _, row := range data {
		for i, value := range row {
			fmt.Fprintf(out, "%-*s", widths[i], value)
			if i < len(row)-1 {
				fmt.Fprint(out, " | ")
			}
		}
		fmt.Fprintln(out)
	}
	fmt.Fprintf(out, "(%d rows)\n", len(data))
}

func (a *app) askDefault(label, fallback string) string {
	if fallback == "" {
		fmt.Fprintf(a.out, "%s: ", label)
	} else {
		fmt.Fprintf(a.out, "%s [%s]: ", label, fallback)
	}
	if !a.in.Scan() {
		return fallback
	}
	text := strings.TrimSpace(a.in.Text())
	if text == "" {
		return fallback
	}
	return text
}

func (a *app) askRequired(label string) string {
	for {
		value := strings.TrimSpace(a.askDefault(label, ""))
		if value != "" {
			return value
		}
		fmt.Fprintln(a.out, "Please enter a value.")
	}
}

func (a *app) askIdentifier(label string) string {
	for {
		value := strings.TrimSpace(a.askRequired(label))
		if validIdentifier(value) {
			return value
		}
		fmt.Fprintln(a.out, "Use letters, numbers, or underscores, and start with a letter or underscore.")
	}
}

func (a *app) askIdentifierList(label string) []string {
	for {
		values, err := parseIdentifierList(a.askRequired(label))
		if err == nil {
			return values
		}
		fmt.Fprintf(a.out, "%v\n", err)
	}
}

func (a *app) askConstraintAction(label string) string {
	actions := map[string]string{
		"":            "",
		"none":        "",
		"cascade":     "CASCADE",
		"restrict":    "RESTRICT",
		"no action":   "NO ACTION",
		"set null":    "SET NULL",
		"set default": "SET DEFAULT",
	}
	for {
		value := strings.ToLower(strings.TrimSpace(a.askDefault(label+" (none, cascade, restrict, no action, set null, set default)", "none")))
		if action, ok := actions[value]; ok {
			return action
		}
		fmt.Fprintln(a.out, "Choose none, cascade, restrict, no action, set null, or set default.")
	}
}

func (a *app) askPrivileges() []string {
	allowed := map[string]string{
		"select":     "SELECT",
		"insert":     "INSERT",
		"update":     "UPDATE",
		"delete":     "DELETE",
		"truncate":   "TRUNCATE",
		"references": "REFERENCES",
		"trigger":    "TRIGGER",
		"all":        "ALL PRIVILEGES",
	}
	for {
		values := splitCSV(a.askRequired("Privileges, comma separated (select, insert, update, delete, truncate, references, trigger, all)"))
		privileges := make([]string, 0, len(values))
		valid := true
		for _, value := range values {
			privilege, ok := allowed[strings.ToLower(value)]
			if !ok {
				fmt.Fprintf(a.out, "Unknown privilege %q.\n", value)
				valid = false
				break
			}
			privileges = append(privileges, privilege)
		}
		if valid && len(privileges) > 0 {
			return privileges
		}
	}
}

func (a *app) askTableName() tableName {
	return a.askTableNameWithPrompt("Table name")
}

func (a *app) askTableNameWithPrompt(prompt string) tableName {
	for {
		value := strings.TrimSpace(a.askRequired(prompt + " (schema.table or table)"))
		table, err := parseTableName(value)
		if err == nil {
			return table
		}
		fmt.Fprintf(a.out, "%v\n", err)
	}
}

func (a *app) askType() string {
	fmt.Fprintln(a.out, "Common types:")
	for i, typ := range commonTypes {
		fmt.Fprintf(a.out, "  %2d. %s\n", i+1, typ)
	}
	for {
		value := strings.TrimSpace(a.askRequired("Column type or number"))
		if n, err := strconv.Atoi(value); err == nil && n >= 1 && n <= len(commonTypes) {
			return commonTypes[n-1]
		}
		if validSQLType(value) {
			return strings.ToUpper(value)
		}
		fmt.Fprintln(a.out, "Enter a simple SQL type such as TEXT, INTEGER, VARCHAR(100), or choose a number.")
	}
}

type tableName struct {
	Schema string
	Name   string
}

func (t tableName) SQL() string {
	if t.Schema == "" {
		return quoteIdent(t.Name)
	}
	return quoteIdent(t.Schema) + "." + quoteIdent(t.Name)
}

func parseTableName(value string) (tableName, error) {
	parts := strings.Split(value, ".")
	if len(parts) > 2 {
		return tableName{}, errors.New("use table or schema.table")
	}
	for _, part := range parts {
		if !validIdentifier(part) {
			return tableName{}, errors.New("table names can only use letters, numbers, and underscores")
		}
	}
	if len(parts) == 1 {
		return tableName{Schema: "public", Name: parts[0]}, nil
	}
	return tableName{Schema: parts[0], Name: parts[1]}, nil
}

func quoteIdent(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func quoteLiteral(value string) string {
	return `'` + strings.ReplaceAll(value, `'`, `''`) + `'`
}

func quoteIdentifiers(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, quoteIdent(value))
	}
	return strings.Join(quoted, ", ")
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parseIdentifierList(value string) ([]string, error) {
	values := splitCSV(value)
	if len(values) == 0 {
		return nil, errors.New("enter at least one identifier")
	}
	for _, value := range values {
		if !validIdentifier(value) {
			return nil, fmt.Errorf("invalid identifier %q", value)
		}
	}
	return values, nil
}

func validIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for i, r := range value {
		if i == 0 && !(unicode.IsLetter(r) || r == '_') {
			return false
		}
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_') {
			return false
		}
	}
	return true
}

func validSQLType(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '(' || r == ')' || r == ',' || unicode.IsSpace(r)) {
			return false
		}
	}
	return true
}

func yes(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "y" || value == "yes" || value == "true" || value == "1"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func looksLikeQuery(sqlText string) bool {
	sqlText = strings.ToLower(strings.TrimSpace(sqlText))
	return strings.HasPrefix(sqlText, "select") ||
		strings.HasPrefix(sqlText, "show") ||
		strings.HasPrefix(sqlText, "with") ||
		strings.HasPrefix(sqlText, "explain")
}
