all: build

build:
	go build -o ./main/main ./main

run:
	./main/main

doc:
	@echo "http://localhost:6060/pkg/github.com/boreq/lainnet/"
	godoc -http=:6060

test:
	go test ./...

proto:
	protoc --proto_path="protocol/message" --go_out="protocol/message" protocol/message/message.proto

clean:
	rm -f ./main/main

.PHONY: build run test proto clean
