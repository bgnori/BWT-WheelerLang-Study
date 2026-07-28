.PHONY: all build test lint clean download-testdata docker-build docker-run

BINARY  := bwtsearch
DATADIR := data
INDEXFILE := $(DATADIR)/moby_dick.idx

all: build

## build: compile the bwtsearch binary
build:
	go build -o $(BINARY) ./cmd/bwtsearch

## test: run all unit tests
test:
	go test -race ./...

## test-verbose: run all unit tests with verbose output
test-verbose:
	go test -race -v ./...

## lint: run go vet
lint:
	go vet ./...

## clean: remove build artifacts and index files
clean:
	rm -f $(BINARY)
	rm -f $(DATADIR)/*.idx

## download-testdata: download Project Gutenberg Moby Dick text (not committed)
download-testdata:
	@mkdir -p $(DATADIR)
	./scripts/download_testdata.sh $(DATADIR)

## build-index: build the FM-index from the downloaded Moby Dick text
build-index: $(DATADIR)/moby_dick.txt
	./$(BINARY) build $(DATADIR)/moby_dick.txt $(INDEXFILE)

## search-demo: run a sample search on the Moby Dick index
search-demo: $(INDEXFILE)
	./$(BINARY) search $(INDEXFILE) "white whale" --limit 10

## compare-demo: compare FM-index vs stdlib for a sample pattern
compare-demo: $(DATADIR)/moby_dick.txt
	./$(BINARY) compare $(DATADIR)/moby_dick.txt "whale"

## docker-build: build the Docker image
docker-build:
	docker compose build

## docker-run: run an interactive bwtsearch container
docker-run:
	docker compose run bwtsearch

help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
