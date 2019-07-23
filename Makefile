BUILD_DIRECTORY=_build
BUILD_FILE=${BUILD_DIRECTORY}/starlight

all: test build

build:
	mkdir -p ${BUILD_DIRECTORY}
	go build -o ${BUILD_FILE} ./cmd/starlight

doc:
	@echo "http://localhost:6060/pkg/github.com/boreq/starlight/"
	@echo "In order to display unexported declarations append ?m=all to an url after"
	@echo "opening docs for a specific package."
	godoc -http=:6060

install-tools:
	go get -v -u honnef.co/go/tools/cmd/staticcheck

analyze: analyze-vet analyze-staticcheck

analyze-vet:
	# go vet
	go vet github.com/boreq/starlight/...

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

words-update:
	cat /usr/share/dict/american-english | egrep '^[a-z]+$$' | egrep '[a-z]{4}' > irc/humanizer/data/words.txt
	@echo "Number of words: $$(cat irc/humanizer/data/words.txt | wc -l)"

words-build:
	statik -src=irc/humanizer/data -dest=irc/humanizer

clean:
	rm -rf ${BUILD_DIRECTORY}

.PHONY: all build doc install-tools analyze analyze-vet analyze-staticcheck test test-verbose test-short bench proto clean
