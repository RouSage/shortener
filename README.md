# Shortener

A simple URL shortener application built with Go, Echo, and PostgreSQL.

## Getting Started

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes. See deployment for notes on how to deploy the project on a live system.

### Prerequisites

- Go 1.24+
- Docker
- sqlc
- golangci-lint
- Air (Optional)

### SQL Queries

SQL queries are implemented using [sqlc](https://sqlc.dev/). Use the `sqlc generate` command to generate the Go code for the SQL queries.

## Makefile

List of available make commands

```bash
make help
```

### Running the application

Set up Git hooks

```bash
sh install-hooks.sh
```

Install dependencies

```bash
go mod tidy
```

Fill out `.env` file with the required environment variables

```bash
cp .env.example .env
```

Run the required services in Docker

```bash
docker compose up -d db valkey
```

Run the application

```bash
make run
or
make watch # run the app with live reload
```

### Build/Run commands

Build the application

```bash
make build
```

Build the application for development

```bash
make build/dev
```

Live reload the application (with Air)

```bash
make watch
```

Clean up binary from the last build:

```bash
make clean
```

### DB Migrations

Create a new migration file

```bash
make migrate/new name={name}
```

Apply all migrations

```bash
make migrate/up
```

Rollback the last migration

```bash
make migrate/down
```

Migrate to a specific version

```bash
make migrate/force version={version}
```

Show the current migration version

```bash
make migrate/version
```

### Quality Control

Audit the application for vulnerabilities, code quality, and dependency issues

```bash
make audit
```

Run all tests, including DB integration tests (made with testcontainers)

```bash
make test
```

Run all tests and display coverage

```bash
make test/cover
```

List direct dependencies that have upgrades available

```bash
make upgradeable
```

Tidy module dependencies and format .go files

```bash
make tidy
```
