UNAME_M := $(shell uname -m)

ifeq ($(UNAME_M), x86_64)
    ARCH := x86_64
else ifeq ($(UNAME_M), aarch64)
    ARCH := arm64
else ifeq ($(UNAME_M), arm64)
    ARCH := arm64
else
    ARCH := $(UNAME_M)
endif


VERSION := $(shell grep -Eo '(v[0-9]+[\.][0-9]+[\.][0-9]+([-a-zA-Z0-9]*)?)' version.go)

.PHONY: build
build:
	go build -o bin/deadcheck github.com/adamdecaf/deadcheck

docker:
	docker build --pull -t adamdecaf/deadcheck:$(VERSION).$(ARCH) -f Dockerfile .

docker-push:
	docker push adamdecaf/deadcheck:$(VERSION).$(ARCH)

docker-manifest:
	docker manifest create \
		adamdecaf/deadcheck:${VERSION} \
		adamdecaf/deadcheck:${VERSION}.x86_64 \
		adamdecaf/deadcheck:${VERSION}.arm64
	docker manifest push adamdecaf/deadcheck:${VERSION}

.PHONY: check
check:
ifeq ($(OS),Windows_NT)
	go test ./...
else
	@wget -O lint-project.sh https://raw.githubusercontent.com/moov-io/infra/master/go/lint-project.sh
	@chmod +x ./lint-project.sh
	COVER_THRESHOLD=0.0 ./lint-project.sh
endif

.PHONY: clean
clean:
	@rm -rf ./bin/ ./tmp/ coverage.txt misspell* staticcheck lint-project.sh

.PHONY: cover-test cover-web
cover-test:
	go test -coverprofile=cover.out ./...
cover-web:
	go tool cover -html=cover.out
