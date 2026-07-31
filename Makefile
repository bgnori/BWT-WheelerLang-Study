.PHONY: all build test lint clean download-testdata download-kenshin download-git download-ecoli prepare-ecoli download-osativa-chr1 prepare-osativa-chr1 download-amazon-small prepare-amazon-small download-amazon-medium prepare-amazon-medium download-amazon-large prepare-amazon-large generate-fake-logs generate-fake-log-apache-common generate-fake-log-apache-error generate-fake-log-syslog generate-fake-log-json generate-fake-log-logfmt docker-build docker-run

BINARY  := bwtsearch
DATADIR := data
INDEXFILE := $(DATADIR)/moby_dick.idx
KENSHIN_INDEXFILE := $(DATADIR)/kenshin.idx
GIT_SRC_DIR := $(DATADIR)/git-src
GIT_INDEXFILE := $(DATADIR)/git.idx
ECOLI_INDEXFILE := $(DATADIR)/ecoli.idx
OSATIVA_CHR1_INDEXFILE := $(DATADIR)/osativa_chr1.idx
AMAZON_SMALL_TEXT := $(DATADIR)/amazon_small.txt
AMAZON_MEDIUM_TEXT := $(DATADIR)/amazon_medium.txt
AMAZON_LARGE_TEXT := $(DATADIR)/amazon_large.txt
FAKE_LOG_SIZE ?= 1M

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
	rm -f $(DATADIR)/*.idx $(DATADIR)/*.saidx

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

## suffixarray-demo: build and search with stdlib suffix array for comparison
suffixarray-demo: $(DATADIR)/moby_dick.txt
	./$(BINARY) build --algo suffixarray $(DATADIR)/moby_dick.txt $(DATADIR)/moby_dick.saidx
	./$(BINARY) search --limit 10 $(DATADIR)/moby_dick.saidx "whale"

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

## suffixarray-demo-kenshin: build and search with stdlib suffix array for 上杉謙信
suffixarray-demo-kenshin: $(DATADIR)/kenshin.txt
	./$(BINARY) build --algo suffixarray $(DATADIR)/kenshin.txt $(DATADIR)/kenshin.saidx
	./$(BINARY) search --limit 5 $(DATADIR)/kenshin.saidx "上杉謙信"

## docker-build: build the Docker image
docker-build:
	docker compose build

## docker-run: run an interactive bwtsearch container
docker-run:
	docker compose run bwtsearch

## download-git: download Git source code (.c/.h files) into data/git-src/
download-git:
	@mkdir -p $(DATADIR)
	./scripts/download_git.sh $(DATADIR)

## download-ecoli: download E. coli K-12 MG1655 genome (FASTA) from NCBI (not committed)
download-ecoli:
	@mkdir -p $(DATADIR)
	./scripts/download_ecoli.sh $(DATADIR)

## prepare-ecoli: convert FASTA to plain-text DNA for FM-index (not committed)
prepare-ecoli: $(DATADIR)/ecoli.fna
	./scripts/prepare_ecoli.sh $(DATADIR)

## download-osativa-chr1: download Oryza sativa chromosome 1 FASTA from NCBI (not committed)
download-osativa-chr1:
	@mkdir -p $(DATADIR)
	./scripts/download_osativa_chr1.sh $(DATADIR)

## prepare-osativa-chr1: convert Oryza sativa chromosome 1 FASTA to plain-text DNA (not committed)
prepare-osativa-chr1: $(DATADIR)/osativa_chr1.fna
	./scripts/prepare_ecoli.sh $(DATADIR) osativa_chr1.fna osativa_chr1.txt

## download-amazon-small: download Kaggle Amazon Laptop Prices Dataset (not committed)
download-amazon-small:
	@mkdir -p $(DATADIR)
	./scripts/download_kaggle_amazon.sh small $(DATADIR)

## prepare-amazon-small: preprocess Kaggle Amazon Laptop Prices Dataset to plain text
prepare-amazon-small:
	./scripts/prepare_kaggle_amazon.sh small $(DATADIR) amazon_small.txt

## download-amazon-medium: download Kaggle Amazon Mobile Dataset (not committed)
download-amazon-medium:
	@mkdir -p $(DATADIR)
	./scripts/download_kaggle_amazon.sh medium $(DATADIR)

## prepare-amazon-medium: preprocess Kaggle Amazon Mobile Dataset to plain text
prepare-amazon-medium:
	./scripts/prepare_kaggle_amazon.sh medium $(DATADIR) amazon_medium.txt

## download-amazon-large: download Kaggle Amazon Product Dataset (100K+) (not committed)
download-amazon-large:
	@mkdir -p $(DATADIR)
	./scripts/download_kaggle_amazon.sh large $(DATADIR)

## prepare-amazon-large: preprocess Kaggle Amazon Product Dataset (100K+) to plain text
prepare-amazon-large:
	./scripts/prepare_kaggle_amazon.sh large $(DATADIR) amazon_large.txt

## generate-fake-logs: generate all 5 fake log datasets (flog + mclogs)
generate-fake-logs: generate-fake-log-apache-common generate-fake-log-apache-error generate-fake-log-syslog generate-fake-log-json generate-fake-log-logfmt

## generate-fake-log-apache-common: generate Apache access log via flog
generate-fake-log-apache-common:
	@mkdir -p $(DATADIR)
	./scripts/generate_fake_logs.sh apache-common $(FAKE_LOG_SIZE) $(DATADIR)

## generate-fake-log-apache-error: generate Apache error log via flog
generate-fake-log-apache-error:
	@mkdir -p $(DATADIR)
	./scripts/generate_fake_logs.sh apache-error $(FAKE_LOG_SIZE) $(DATADIR)

## generate-fake-log-syslog: generate Syslog (RFC3164) via flog
generate-fake-log-syslog:
	@mkdir -p $(DATADIR)
	./scripts/generate_fake_logs.sh syslog $(FAKE_LOG_SIZE) $(DATADIR)

## generate-fake-log-json: generate JSON logs via mclogs
generate-fake-log-json:
	@mkdir -p $(DATADIR)
	./scripts/generate_fake_logs.sh json $(FAKE_LOG_SIZE) $(DATADIR)

## generate-fake-log-logfmt: generate logfmt logs via mclogs
generate-fake-log-logfmt:
	@mkdir -p $(DATADIR)
	./scripts/generate_fake_logs.sh logfmt $(FAKE_LOG_SIZE) $(DATADIR)

## build-index-git: build the FM-index from the Git source files
build-index-git: $(GIT_SRC_DIR)
	find $(GIT_SRC_DIR) -type f \( -name "*.c" -o -name "*.h" \) | sort | \
	    xargs ./$(BINARY) build-multi $(GIT_INDEXFILE)

## search-demo-git: run a sample search on the Git source index
search-demo-git: $(GIT_INDEXFILE)
	./$(BINARY) search --limit 10 $(GIT_INDEXFILE) "commit"

## suffixarray-demo-git: build and search with stdlib suffix array for Git source
suffixarray-demo-git: $(GIT_SRC_DIR)
	find $(GIT_SRC_DIR) -type f \( -name "*.c" -o -name "*.h" \) | sort | \
	    xargs ./$(BINARY) build-multi --algo suffixarray $(DATADIR)/git.saidx
	./$(BINARY) search --limit 10 $(DATADIR)/git.saidx "commit"

## build-index-ecoli: build the FM-index from the E. coli genome text (SA-IS recommended for large input)
build-index-ecoli: $(DATADIR)/ecoli.txt
	./$(BINARY) build --algo sais $(DATADIR)/ecoli.txt $(ECOLI_INDEXFILE)

## search-demo-ecoli: run a sample search on the E. coli genome index
search-demo-ecoli: $(ECOLI_INDEXFILE)
	./$(BINARY) search --limit 10 $(ECOLI_INDEXFILE) "ATGAAACGC"

## suffixarray-demo-ecoli: build and search with stdlib suffix array for E. coli genome
suffixarray-demo-ecoli: $(DATADIR)/ecoli.txt
	./$(BINARY) build --algo suffixarray $(DATADIR)/ecoli.txt $(DATADIR)/ecoli.saidx
	./$(BINARY) search --limit 10 $(DATADIR)/ecoli.saidx "ATGAAACGC"

## build-index-osativa-chr1: build FM-index from Oryza sativa chr1 genome text
build-index-osativa-chr1: $(DATADIR)/osativa_chr1.txt
	./$(BINARY) build --algo sais $(DATADIR)/osativa_chr1.txt $(OSATIVA_CHR1_INDEXFILE)

## build-index-amazon-small: build FM-index from preprocessed Amazon small dataset
build-index-amazon-small: $(AMAZON_SMALL_TEXT)
	./$(BINARY) build --algo sais $(AMAZON_SMALL_TEXT) $(DATADIR)/amazon_small.idx

## build-index-amazon-medium: build FM-index from preprocessed Amazon medium dataset
build-index-amazon-medium: $(AMAZON_MEDIUM_TEXT)
	./$(BINARY) build --algo sais $(AMAZON_MEDIUM_TEXT) $(DATADIR)/amazon_medium.idx

## build-index-amazon-large: build FM-index from preprocessed Amazon large dataset
build-index-amazon-large: $(AMAZON_LARGE_TEXT)
	./$(BINARY) build --algo sais $(AMAZON_LARGE_TEXT) $(DATADIR)/amazon_large.idx

## search-demo-osativa-chr1: run a sample search on the Oryza sativa chr1 genome index
search-demo-osativa-chr1: $(OSATIVA_CHR1_INDEXFILE)
	./$(BINARY) search --limit 10 $(OSATIVA_CHR1_INDEXFILE) "ATGGCG"

help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
