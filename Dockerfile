FROM golang:1.24.3-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main cmd/api/main.go

FROM alpine:3.21.3 AS prod

WORKDIR /app

COPY --from=build /app/main /app/main

EXPOSE ${PORT}

CMD ["./main"]
