.PHONY: build test install lint clean dist

VERSION ?= $(shell scripts/version.sh)
PLATFORMS := darwin/arm64 darwin/amd64 linux/amd64 linux/arm64

build:
	go build -o bin/commando ./cmd/commando

test:
	go test ./...

install:
	go install ./cmd/commando

lint:
	gofmt -l . | (! grep .)
	go vet ./...

# Cross-compiled release archives and checksums in dist/.
dist:
	rm -rf dist && mkdir -p dist
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; name=commando_$(VERSION)_$${os}_$${arch}; \
		echo "building $$name"; \
		mkdir -p dist/$$name && \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath \
			-ldflags "-s -w -X main.version=$(VERSION)" -o dist/$$name/commando ./cmd/commando && \
		cp LICENSE README.md dist/$$name/ && \
		tar -C dist -czf dist/$$name.tar.gz $$name && rm -rf dist/$$name || exit 1; \
	done
	cd dist && (command -v sha256sum >/dev/null && sha256sum *.tar.gz || shasum -a 256 *.tar.gz) > checksums.txt

clean:
	rm -rf bin dist
