# Postgo

Postgo is a guided Go command line application for people who do not want to
memorize SQL syntax. It connects to PostgreSQL, lets the user choose simple
commands like `create-table`, `insert-row`, or `list-tables`, asks for the
needed inputs, shows the SQL it generated, and only runs it after confirmation.

## Hobby Project Notice

Postgo is a hobby build and learning project. It is not a professional database
administration tool, has not been security-audited, and should not be used for
production or professional environments. Use it for local practice, experiments,
and learning PostgreSQL command patterns.

## Run Locally

```sh
go run .
```

Postgo reads these environment variables as connection defaults:

```sh
PGHOST=localhost
PGPORT=5432
PGUSER=postgres
PGPASSWORD=
PGDATABASE=postgres
PGSSLMODE=disable
```

You can also enter or override the values interactively when the app starts.

## Run With Docker

```sh
docker build -t postgo .
docker run -it --rm postgo
```

To pass connection defaults:

```sh
docker run -it --rm \
  -e PGHOST=host.docker.internal \
  -e PGPORT=5432 \
  -e PGUSER=postgres \
  -e PGPASSWORD=password \
  -e PGDATABASE=postgres \
  postgo
```

## Commands

Type `help` inside the CLI to see the current command list. The application
includes guided commands for:

- Connecting and checking status: `connect`, `status`
- Databases: `list-db`, `create-db`, `drop-db`
- Schemas: `list-schemas`, `create-schema`, `drop-schema`
- Tables: `list-tables`, `describe-table`, `show-columns`, `create-table`,
  `drop-table`, `truncate-table`, `rename-table`
- Columns: `add-column`, `drop-column`, `rename-column`, `alter-column-type`
- Constraints: `set-not-null`, `drop-not-null`, `add-primary-key`,
  `add-foreign-key`, `drop-constraint`, `list-constraints`
- Indexes: `create-index`, `drop-index`, `list-indexes`
- Data: `insert-row`, `select-rows`, `update-rows`, `delete-rows`, `count-rows`
- Views: `create-view`, `drop-view`, `list-views`
- Roles and access: `list-roles`, `create-role`, `grant-table`, `revoke-table`
- Utilities: `analyze-table`, `vacuum-table`, `backup-table-as`,
  `explain-query`, `raw-sql`

Most commands have short aliases, such as `h` for `help`, `ct` for
`create-table`, `insert` for `insert-row`, and `q` for `quit`.

## Safety Notes

Postgo quotes database, schema, table, column, index, role, and constraint
identifiers it builds from prompts. Commands that change data or schema print
the generated SQL and ask for confirmation before executing.

For flexible clauses like `WHERE` and advanced `SELECT` queries, the user still
types the SQL fragment because those clauses can be very application-specific.
