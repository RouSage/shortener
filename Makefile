# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	echo 'Usage:'
	sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

.PHONY: confirm
confirm:
	echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

.PHONY: no-dirty
no-dirty:
	test -z "$(shell git status --porcelain)"

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## audit: run quality control checks
.PHONY: audit
audit:
	@echo "Checking module dependencies..."
	go mod tidy -diff
	go mod verify
	test -z "$(shell gofmt -l .)"
	echo "Vetting code..."
	go vet ./...
	golangci-lint run ./...
	go tool govulncheck ./...

## test: run all tests
.PHONY: test
test:
	echo "Testing..."
	go test -race ./...

## test/cover: run all tests and display coverage
.PHONY: test/cover
test/cover:
	go test -v -race -coverprofile=/tmp/coverage.out ./...
	go tool cover -html=/tmp/coverage.out

## upgradeable: list direct dependencies that have upgrades available
.PHONY: upgradeable
upgradeable:
	go list -u -f '{{if (and (not (or .Main .Indirect)) .Update)}}{{.Path}}: {{.Version}} -> {{.Update.Version}}{{end}}' -m all

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## tidy: tidy modfiles and format .go files
.PHONY: tidy
tidy:
	echo "Tidying module dependencies..."
	go mod tidy
	echo "Formatting code..."
	go fmt ./...

## build: build the application
.PHONY: build
build: no-dirty audit test
	echo "Building..."
	go build -ldflags='-s' -o ./bin/api ./cmd/api/main.go
	GOOS=linux GOARCH=amd64 go build -ldflags='-s' -o ./bin/linux_amd64/api ./cmd/api/main.go

## build/dev: build the application for development
.PHONY: build/dev
build/dev:
	go build -ldflags='-s' -o ./bin/api ./cmd/api/main.go

## run: run the application
.PHONY: run
run:
	go run cmd/api/main.go

## clean: clean the binary
.PHONY: clean
clean:
	echo "Cleaning..."
	rm -rf bin/

## watch: live Reload
.PHONY: watch
watch:
	if command -v air > /dev/null; then \
            air; \
            echo "Watching...";\
        else \
            read -p "Go's 'air' is not installed on your machine. Do you want to install it? [Y/n] " choice; \
            if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
                go install github.com/air-verse/air@latest; \
                air; \
                echo "Watching...";\
            else \
                echo "You chose not to install air. Exiting..."; \
                exit 1; \
            fi; \
        fi

## swagger: generate swagger docs
.PHONY: swagger
swagger:
	go tool swag i -g internal/server/routes.go

# ==================================================================================== #
# DATABASE
# ==================================================================================== #

## migrate/new name=$1: create a new migration file
.PHONY: migrate/new
migrate/new:
		echo 'Creating migration files for ${name}...'
		go run -tags 'pgx5' github.com/golang-migrate/migrate/v4/cmd/migrate@latest \
			create -seq -ext sql -dir ./internal/database/migrations ${name}

## migrate/up: apply all migrations
.PHONY: migrate/up
migrate/up: confirm
	set -a; \
	if [ -f .env ]; then source .env; fi; \
	set +a; \
	go run -tags 'pgx5' github.com/golang-migrate/migrate/v4/cmd/migrate@latest \
		-database "pgx5://$$DB_USERNAME:$$DB_PASSWORD@$$DB_HOST:$$DB_PORT/$$DB_DATABASE?sslmode=disable&search_path=$$DB_SCHEMA" \
		-path ./internal/database/migrations \
		up

## migrate/down: rollback the last migration
.PHONY: migrate/down
migrate/down: confirm
	set -a; \
	if [ -f .env ]; then source .env; fi; \
	set +a; \
	go run -tags 'pgx5' github.com/golang-migrate/migrate/v4/cmd/migrate@latest \
		-database "pgx5://$$DB_USERNAME:$$DB_PASSWORD@$$DB_HOST:$$DB_PORT/$$DB_DATABASE?sslmode=disable&search_path=$$DB_SCHEMA" \
		-path ./internal/database/migrations \
		down 1

## migrate/force version=$1: migrate to a specific version
.PHONY: migrate/force
migrate/force: confirm
	set -a; \
	if [ -f .env ]; then source .env; fi; \
	set +a; \
	go run -tags 'pgx5' github.com/golang-migrate/migrate/v4/cmd/migrate@latest \
		-database "pgx5://$$DB_USERNAME:$$DB_PASSWORD@$$DB_HOST:$$DB_PORT/$$DB_DATABASE?sslmode=disable&search_path=$$DB_SCHEMA" \
		-path ./internal/database/migrations \
		force ${version}

## migrate/version: show the current migration version
.PHONY: migrate/version
migrate/version:
	set -a; \
	if [ -f .env ]; then source .env; fi; \
	set +a; \
	go run -tags 'pgx5' github.com/golang-migrate/migrate/v4/cmd/migrate@latest \
		-database "pgx5://$$DB_USERNAME:$$DB_PASSWORD@$$DB_HOST:$$DB_PORT/$$DB_DATABASE?sslmode=disable&search_path=$$DB_SCHEMA" \
		-path ./internal/database/migrations \
		version
