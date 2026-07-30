.PHONY: all build test lint clean download-testdata download-kenshin docker-build docker-run

BINARY  := bwtsearch
DATADIR := data
INDEXFILE := $(DATADIR)/moby_dick.idx
KENSHIN_INDEXFILE := $(DATADIR)/kenshin.idx

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
	./$(BINARY) search --limit 10 $(INDEXFILE) "white whale"

## compare-demo: compare FM-index vs stdlib for a sample pattern
compare-demo: $(DATADIR)/moby_dick.txt
	./$(BINARY) compare $(DATADIR)/moby_dick.txt "whale"

## download-kenshin: download Aozora Bunko 上杉謙信 and convert to UTF-8 (not committed)
download-kenshin:
	@mkdir -p $(DATADIR)
	./scripts/download_kenshin.sh $(DATADIR)

## build-index-kenshin: build the FM-index from the 上杉謙信 text
build-index-kenshin: $(DATADIR)/kenshin.txt
	./$(BINARY) build $(DATADIR)/kenshin.txt $(KENSHIN_INDEXFILE)

## search-demo-kenshin: run a sample search on the 上杉謙信 index
search-demo-kenshin: $(KENSHIN_INDEXFILE)
	./$(BINARY) search --limit 5 $(KENSHIN_INDEXFILE) "上杉謙信"

## compare-demo-kenshin: compare FM-index vs stdlib for a sample pattern
compare-demo-kenshin: $(DATADIR)/kenshin.txt
	./$(BINARY) compare $(DATADIR)/kenshin.txt "上杉謙信"

## docker-build: build the Docker image
docker-build:
	docker compose build

## docker-run: run an interactive bwtsearch container
docker-run:
	docker compose run bwtsearch

help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
