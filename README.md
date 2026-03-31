# Shortener

A simple URL shortener application built with Go, Echo, and PostgreSQL.

## Getting Started

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes. See deployment for notes on how to deploy the project on a live system.

### Prerequisites

- Go 1.25+
- Docker
- sqlc
- golangci-lint
- Air (Optional)

### SQL Queries

SQL queries are implemented using [sqlc](https://sqlc.dev/). Use the `sqlc generate` command to generate the Go code for the SQL queries.

## Just

List of available just commands

```bash
just
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
docker compose up -d db valkey otel-collector
# or
just docker-up
```

Run the application

```bash
just run
# or
just watch # run the app with live reload
```

### Build/Run commands

Build the application

```bash
just build
```

Build the application for development

```bash
just build-dev
```

Live reload the application (with Air)

```bash
just watch
```

Clean up binary from the last build:

```bash
just clean
```

### DB Migrations

Create a new migration file

```bash
just migrate-new name
```

Apply all migrations

```bash
just migrate-up
```

Rollback the last migration

```bash
just migrate-down
```

Migrate to a specific version

```bash
just migrate-force version
```

Show the current migration version

```bash
just migrate-version
```

### Quality Control

Audit the application for vulnerabilities, code quality, and dependency issues

```bash
just audit
```

Run all tests, including DB integration tests (made with testcontainers)

```bash
just test
```

Run all tests and display coverage

```bash
just test-cover
```

List direct dependencies that have upgrades available

```bash
just upgradeable
```

Tidy module dependencies and format .go files

```bash
just tidy
```
