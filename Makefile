all: build

build:
	mkdir -p _build
	go build -o ./_build/starlight ./cmd/starlight

doc:
	@echo "http://localhost:6060/pkg/github.com/boreq/starlight/"
	@echo "In order to display unexported declarations append ?m=all to an url after"
	@echo "opening docs for a specific package."
	godoc -http=:6060

install-tools:
	go get -u honnef.co/go/tools/cmd/unused
	go get -u honnef.co/go/tools/cmd/staticcheck

analyze: analyze-vet analyze-unused analyze-staticcheck

analyze-vet:
	go vet github.com/boreq/starlight/...

analyze-unused:
	# https://github.com/dominikh/go-tools/tree/master/cmd/unused
	unused github.com/boreq/starlight/...

analyze-staticcheck:
	# https://github.com/dominikh/go-tools/tree/master/cmd/staticcheck
	staticcheck github.com/boreq/starlight/...

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

.PHONY: all build doc install-tools analyze analyze-vet analyze-unused analyze-staticcheck test test-verbose test-short bench proto clean
