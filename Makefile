.PHONY: build test install lint clean

build:
	go build -o bin/commando ./cmd/commando

test:
	go test ./...

install:
	go install ./cmd/commando

lint:
	gofmt -l . | (! grep .)
	go vet ./...

clean:
	rm -rf bin
