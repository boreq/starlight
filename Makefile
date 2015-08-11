all: build

build:
	go build -o ./main/main ./main

run:
	./main/main

test:
	go test ./...

proto:
	protoc --proto_path="protocol/proto" --go_out="protocol/message" protocol/proto/message.proto

clean:
	rm -f ./main/main

.PHONY: build run test proto clean
