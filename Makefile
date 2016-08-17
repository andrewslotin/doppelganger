VERSION := 1.1.1
BUILDDATE :=$(shell date +%F)
LDFLAGS := -X 'main.Version=$(VERSION)' -X 'main.BuildDate=$(BUILDDATE)'

build: test doppelganger

release: OS=linux
release: ARCH=amd64
release: doppelganger-$(VERSION)_$(OS)_$(ARCH).tar.gz

doppelganger-$(VERSION)_$(OS)_$(ARCH).tar.gz: doppelganger
	goupx doppelganger
	tar czf doppelganger-$(VERSION)_$(OS)_$(ARCH).tar.gz assets/ templates/ doppelganger

SOURCES := $(shell find . \( -name '*.go' -and -not -name '*_test.go' \))
doppelganger: $(SOURCES)
	@echo "Building v$(VERSION)"
	GOOS=$(OS) GOARCH=$(ARCH) go build -ldflags "$(LDFLAGS)" -o doppelganger

clean:
	go clean ./...
	rm -rf doppelganger-$(VERSION)_*.tar.gz

PACKAGES := $$(go list ./... | grep -v /vendor/ )
test:
	go test $(PACKAGES)

.PHONY: build test release clean
