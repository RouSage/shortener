set shell := ["zsh", "-cu"]
set dotenv-required := true
set dotenv-load := true

default:
    @just --list --unsorted

[group('help')]
_confirm:
    @echo -n 'Are you sure? [y/N] ' && read ans && [ ${ans:-N} = y ]

[group('help')]
_no-dirty:
    test -z "$(shell git status --porcelain)"

# list direct dependencies that have upgrades available
[group('help')]
upgradeable:
    @go list -u -f "$(cat .go-list-upgradeable.tmpl)" -m all

# run quality control checks
[group('lint')]
audit:
    @echo "Checking module dependencies..."
    go mod tidy -diff
    go mod verify
    test -z "$(gofmt -l .)"
    @echo "Vetting code..."
    go vet ./...
    golangci-lint run ./...
    go tool govulncheck ./...

# run all tests
[group('test')]
test:
    @echo "Testing..."
    go test -race ./...

# run all tests and display coverage
[group('test')]
test-cover:
    go test -v -race -coverprofile=/tmp/coverage.out ./...
    go tool cover -html=/tmp/coverage.out

# run the application services in docker
[group('dev')]
docker-up:
    docker compose up -d db valkey jaeger

# stop the application services in docker
[group('dev')]
docker-down:
    docker compose down

# tidy modfiles and format .go files
[group('dev')]
tidy:
    @echo "Tidying module dependencies..."
    go mod tidy
    @echo "Formatting code..."
    go fmt ./...

# build the application
[group('build')]
[group('dev')]
build: _no-dirty audit test
    @echo "Building..."
    go build -ldflags='-s' -o ./bin/api ./cmd/api/main.go
    GOOS=linux GOARCH=amd64 go build -ldflags='-s' -o ./bin/linux_amd64/api ./cmd/api/main.go

# build the application for development
[group('build')]
[group('dev')]
build-dev:
    go build -ldflags='-s' -o ./bin/api ./cmd/api/main.go

# run the application
[group('dev')]
run:
    go run cmd/api/main.go

# clean the binary
[group('build')]
[group('dev')]
clean:
    @echo "Cleaning..."
    rm -rf bin/

# live Reload
[group('dev')]
watch:
    #!/usr/bin/env bash
    if command -v air > /dev/null; then
        air
        @echo "Watching..."
    else
        read -p "Go's 'air' is not installed on your machine. Do you want to install it? [Y/n] " choice
        if [[ "$choice" != "n" && "$choice" != "N" ]]; then
            go install github.com/air-verse/air@latest
            air
            @echo "Watching..."
        else
            @echo "You chose not to install air. Exiting..."
            exit 1
        fi
    fi

# create a new migration file
[group('db')]
migrate-new NAME:
    @echo 'Creating migration files for {{ NAME }}...'
    go run -tags 'pgx5' github.com/golang-migrate/migrate/v4/cmd/migrate@latest \
    	create -seq -ext sql -dir ./internal/database/migrations {{ NAME }}

# apply all migrations
[group('db')]
migrate-up: _confirm
    @go run -tags 'pgx5' github.com/golang-migrate/migrate/v4/cmd/migrate@latest \
    	-database "pgx5://$DB_USERNAME:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_DATABASE?sslmode=disable&search_path=$DB_SCHEMA" \
    	-path ./internal/database/migrations \
    	up

# rollback the last migration
[group('db')]
migrate-down: _confirm
    @go run -tags 'pgx5' github.com/golang-migrate/migrate/v4/cmd/migrate@latest \
    	-database "pgx5://$DB_USERNAME:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_DATABASE?sslmode=disable&search_path=$DB_SCHEMA" \
    	-path ./internal/database/migrations \
    	down 1

# migrate to a specific version
[group('db')]
migrate-force VERSION: _confirm
    @go run -tags 'pgx5' github.com/golang-migrate/migrate/v4/cmd/migrate@latest \
    	-database "pgx5://$DB_USERNAME:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_DATABASE?sslmode=disable&search_path=$DB_SCHEMA" \
    	-path ./internal/database/migrations \
    	force {{ VERSION }}

# show the current migration version
[group('db')]
migrate-version:
    @go run -tags 'pgx5' github.com/golang-migrate/migrate/v4/cmd/migrate@latest \
    	-database "pgx5://$DB_USERNAME:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_DATABASE?sslmode=disable&search_path=$DB_SCHEMA" \
    	-path ./internal/database/migrations \
    	version

# generate swagger docs
[group('docs')]
swagger-generate:
    go tool swag i -g internal/server/routes.go

# format the swag comments
[group('docs')]
swagger-fmt:
    go tool swag fmt
