VERSION = 1.0.0a
LDFLAGS = -X main.VERSION=0.0.1a -X main.BUILD_DATE=$(shell date +%F)

build: doppelganger

doppelganger:
	@echo "Building v$(VERSION)"
	go build -ldflags "$(LDFLAGS)" -o doppelganger

clean:
	go clean ./...
