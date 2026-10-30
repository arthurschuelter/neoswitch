.PHONY: run test up down
run:
	go run ./cmd/main.go

test:
	go test ./...