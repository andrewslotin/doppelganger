VERSION = 1.0.2
LDFLAGS = -X main.Version=$(VERSION) -X main.BuildDate=$(shell date +%F)

build: 
	@echo "Building v$(VERSION)"
	GOOS=$(OS) GOARCH=$(ARCH) go build -ldflags "$(LDFLAGS)" -o doppelganger

release: OS=linux
release: ARCH=amd64
release: doppelganger-$(VERSION)_$(OS)_$(ARCH).tar.gz

doppelganger-$(VERSION)_$(OS)_$(ARCH).tar.gz: build
	goupx doppelganger
	tar czf doppelganger-$(VERSION)_$(OS)_$(ARCH).tar.gz assets/ templates/ doppelganger

clean:
	go clean ./...
	rm doppelganger-$(VERSION)_*.tar.gz 2>/dev/null || true

.PHONY: clean
