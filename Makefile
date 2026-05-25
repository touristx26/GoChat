.PHONY: build build-server build-client run-server run-client clean

build: build-server build-client

build-server:
	go build -o bin/server ./cmd/server

build-client:
	go build -o bin/client ./cmd/client

run-server:
	go run ./cmd/server

run-client:
	go run ./cmd/client

clean:
	rm -rf bin/
