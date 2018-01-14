all: build

build:
	go build -o ./main/main ./main

run:
	./main/main

doc:
	@echo "http://localhost:6060/pkg/github.com/boreq/starlight/"
	godoc -http=:6060

test:
	go test ./...

test-verbose:
	go test -v ./...

test-short:
	go test -short ./...

bench:
	go test -v -run=XXX -bench=. ./...

proto:
	protoc --proto_path="protocol/message" --go_out="protocol/message" protocol/message/message.proto

clean:
	rm -f ./main/main

.PHONY: all build run doc test test-verbose test-short bench proto clean
