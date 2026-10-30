.PHONY: run test up down
run:
	go run ./cmd/neoswitch/main.go

test:
	go test ./...