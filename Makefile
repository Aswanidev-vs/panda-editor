BINARY_NAME=panda.exe
CHERRY_DIR=cherry

.PHONY: all build cherry test test-cherry test-all vet release run clean

all: build

build:
	go build -o $(BINARY_NAME) .

cherry:
	go -C $(CHERRY_DIR) build ./...

test:
	go test ./...

test-cherry:
	go -C $(CHERRY_DIR) test ./... -count=1

test-all: test test-cherry vet

vet:
	go vet ./...
	go -C $(CHERRY_DIR) vet ./...

release:
	go build -ldflags="-s -w" -o $(BINARY_NAME) .

run: build
	./$(BINARY_NAME)

clean:
	rm -f $(BINARY_NAME)
