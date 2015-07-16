all: build

build:
	go build -o ./main/main ./main

run:
	./main/main

clean:
	rm -f ./main/main

.PHONY: build run clean
