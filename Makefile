.PHONY: all build test lint clean download-moby-dick download-testdata download-kenshin download-git download-ecoli prepare-ecoli download-osativa download-osativa-chr1 prepare-osativa-chr1 prepare-osativa-all download-amazon-small prepare-amazon-small download-amazon-medium prepare-amazon-medium download-amazon-large prepare-amazon-large build-index-moby-dick build-index build-search-demo search-demo suffixarray-demo-moby-dick suffixarray-demo generate-fake-logs generate-fake-log-apache-common generate-fake-log-apache-error generate-fake-log-syslog generate-fake-log-json generate-fake-log-logfmt docker-build docker-run build-index-osativa-all-suffixarray suffixarray-demo-osativa-all time-build-index-osativa-all-sais time-build-index-osativa-all-suffixarray time-search-osativa-all-fm time-search-osativa-all-suffixarray bench-osativa-all
.PHONY: time-build-index-moby-dick-sais time-search-moby-dick-fm bench-moby-dick time-build-index-kenshin-sais time-search-kenshin-fm bench-kenshin time-build-index-git-sais time-search-git-fm bench-git time-build-index-ecoli-sais time-search-ecoli-fm bench-ecoli time-build-index-osativa-chr1-sais time-search-osativa-chr1-fm bench-osativa-chr1 time-build-index-amazon-small-sais time-search-amazon-small-fm bench-amazon-small time-build-index-amazon-medium-sais time-search-amazon-medium-fm bench-amazon-medium time-build-index-amazon-large-sais time-search-amazon-large-fm bench-amazon-large time-build-index-fake-log-apache-common-sais time-search-fake-log-apache-common-fm bench-fake-log-apache-common time-build-index-fake-log-apache-error-sais time-search-fake-log-apache-error-fm bench-fake-log-apache-error time-build-index-fake-log-syslog-sais time-search-fake-log-syslog-fm bench-fake-log-syslog time-build-index-fake-log-json-sais time-search-fake-log-json-fm bench-fake-log-json time-build-index-fake-log-logfmt-sais time-search-fake-log-logfmt-fm bench-fake-log-logfmt bench-fake-logs bench-all-datasets bench-all-datasets-local bench-all-datasets-external

BINARY  := bwtsearch
DATADIR := data
INDEXFILE := $(DATADIR)/moby_dick.idx
KENSHIN_INDEXFILE := $(DATADIR)/kenshin.idx
GIT_SRC_DIR := $(DATADIR)/git-src
GIT_INDEXFILE := $(DATADIR)/git.idx
ECOLI_INDEXFILE := $(DATADIR)/ecoli.idx
OSATIVA_FASTA := $(DATADIR)/osativa.fna
OSATIVA_CHR1_INDEXFILE := $(DATADIR)/osativa_chr1.idx
OSATIVA_ALL_TEXT := $(DATADIR)/osativa_all.txt
OSATIVA_ALL_INDEXFILE := $(DATADIR)/osativa_all.idx
OSATIVA_ALL_SA_INDEXFILE := $(DATADIR)/osativa_all.saidx
OSATIVA_CHR1_SELECTOR ?= AP014957[.]1
OSATIVA_BENCH_QUERY ?= ATGGCG
MOBY_BENCH_QUERY ?= white whale
KENSHIN_BENCH_QUERY ?= 上杉謙信
GIT_BENCH_QUERY ?= commit
ECOLI_BENCH_QUERY ?= ATGAAACGC
OSATIVA_CHR1_BENCH_QUERY ?= ATGGCG
AMAZON_BENCH_QUERY ?= laptop
FAKE_LOG_APACHE_COMMON_BENCH_QUERY ?= GET
FAKE_LOG_APACHE_ERROR_BENCH_QUERY ?= error
FAKE_LOG_SYSLOG_BENCH_QUERY ?= INFO
FAKE_LOG_JSON_BENCH_QUERY ?= level
FAKE_LOG_LOGFMT_BENCH_QUERY ?= level
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

## download-moby-dick: download Project Gutenberg Moby Dick text (not committed)
download-moby-dick:
	@mkdir -p $(DATADIR)
	./scripts/download_testdata.sh $(DATADIR)

## backward compatibility alias for the previous name
download-testdata: download-moby-dick

## build-index-moby-dick: build the FM-index from the downloaded Moby Dick text
build-index-moby-dick: $(DATADIR)/moby_dick.txt
	./$(BINARY) build $(DATADIR)/moby_dick.txt $(INDEXFILE)

## backward compatibility alias for the previous name
build-index: build-index-moby-dick

## search-demo-moby-dick: run a sample search on the Moby Dick index
search-demo-moby-dick: $(INDEXFILE)
	./$(BINARY) search --limit 10 $(INDEXFILE) "white whale"

## backward compatibility alias for the previous name
search-demo: search-demo-moby-dick

## suffixarray-demo-moby-dick: build and search with stdlib suffix array for comparison
suffixarray-demo-moby-dick: $(DATADIR)/moby_dick.txt
	./$(BINARY) build --algo suffixarray $(DATADIR)/moby_dick.txt $(DATADIR)/moby_dick.saidx
	./$(BINARY) search --limit 10 $(DATADIR)/moby_dick.saidx "whale"

## backward compatibility alias for the previous name
suffixarray-demo: suffixarray-demo-moby-dick

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

## download-osativa: download Oryza sativa genome FASTA from NCBI (not committed)
download-osativa: $(OSATIVA_FASTA)

$(OSATIVA_FASTA):
	@mkdir -p $(DATADIR)
	./scripts/download_osativa.sh $(DATADIR)

## download-osativa-chr1: backward compatible alias for downloading Oryza sativa genome FASTA
download-osativa-chr1: download-osativa

## prepare-osativa-chr1: extract Oryza sativa chromosome 1 from the genome FASTA as plain-text DNA
prepare-osativa-chr1: $(DATADIR)/osativa_chr1.txt

$(DATADIR)/osativa_chr1.txt: $(OSATIVA_FASTA)
	./scripts/prepare_fasta_records.sh $(DATADIR) osativa.fna osativa_chr1.txt '$(OSATIVA_CHR1_SELECTOR)'

## prepare-osativa-all: convert all Oryza sativa FASTA records to plain-text DNA
prepare-osativa-all: $(OSATIVA_ALL_TEXT)

$(OSATIVA_ALL_TEXT): $(OSATIVA_FASTA)
	./scripts/prepare_fasta_records.sh $(DATADIR) osativa.fna osativa_all.txt '.'

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
build-index-osativa-chr1: $(OSATIVA_CHR1_INDEXFILE)

$(OSATIVA_CHR1_INDEXFILE): $(DATADIR)/osativa_chr1.txt
	./$(BINARY) build --algo sais $(DATADIR)/osativa_chr1.txt $(OSATIVA_CHR1_INDEXFILE)

## build-index-osativa-all: build FM-index from all Oryza sativa chromosomes
build-index-osativa-all: $(OSATIVA_ALL_INDEXFILE)

$(OSATIVA_ALL_INDEXFILE): $(OSATIVA_ALL_TEXT)
	./$(BINARY) build --algo sais $(OSATIVA_ALL_TEXT) $(OSATIVA_ALL_INDEXFILE)

## build-index-osativa-all-suffixarray: build stdlib suffix-array index from all Oryza sativa chromosomes
build-index-osativa-all-suffixarray: $(OSATIVA_ALL_SA_INDEXFILE)

$(OSATIVA_ALL_SA_INDEXFILE): $(OSATIVA_ALL_TEXT)
	./$(BINARY) build --algo suffixarray $(OSATIVA_ALL_TEXT) $(OSATIVA_ALL_SA_INDEXFILE)

## suffixarray-demo-osativa-all: build and search with stdlib suffix array for all Oryza sativa chromosomes
suffixarray-demo-osativa-all: $(OSATIVA_ALL_SA_INDEXFILE)
	./$(BINARY) search --limit 10 $(OSATIVA_ALL_SA_INDEXFILE) "ATGGCG"

## time-build-index-osativa-all-sais: measure build time for FM-index (SA-IS) on all Oryza sativa chromosomes
time-build-index-osativa-all-sais: $(OSATIVA_ALL_TEXT)
	@tmp=$$(mktemp); \
	start=$$(date +%s); \
	/usr/bin/time -f '%M' -o $$tmp ./$(BINARY) build --algo sais $(OSATIVA_ALL_TEXT) $(OSATIVA_ALL_INDEXFILE); \
	end=$$(date +%s); \
	peak_kb=$$(cat $$tmp); \
	index_bytes=$$(wc -c < $(OSATIVA_ALL_INDEXFILE)); \
	rm -f $$tmp; \
	echo "Elapsed (FM/SAIS): $$((end-start)) sec, peak_rss_kb=$$peak_kb, index_bytes=$$index_bytes"

## time-build-index-osativa-all-suffixarray: measure build time for stdlib suffix array on all Oryza sativa chromosomes
time-build-index-osativa-all-suffixarray: $(OSATIVA_ALL_TEXT)
	@tmp=$$(mktemp); \
	start=$$(date +%s); \
	/usr/bin/time -f '%M' -o $$tmp ./$(BINARY) build --algo suffixarray $(OSATIVA_ALL_TEXT) $(OSATIVA_ALL_SA_INDEXFILE); \
	end=$$(date +%s); \
	peak_kb=$$(cat $$tmp); \
	index_bytes=$$(wc -c < $(OSATIVA_ALL_SA_INDEXFILE)); \
	rm -f $$tmp; \
	echo "Elapsed (Stdlib/SuffixArray): $$((end-start)) sec, peak_rss_kb=$$peak_kb, index_bytes=$$index_bytes"

## time-search-osativa-all-fm: measure search time on all Oryza sativa FM-index
time-search-osativa-all-fm: $(OSATIVA_ALL_INDEXFILE)
	@tmp=$$(mktemp); \
	start=$$(date +%s); \
	/usr/bin/time -f '%M' -o $$tmp ./$(BINARY) search --limit 10 $(OSATIVA_ALL_INDEXFILE) "$(OSATIVA_BENCH_QUERY)" > /dev/null; \
	end=$$(date +%s); \
	peak_kb=$$(cat $$tmp); \
	rm -f $$tmp; \
	echo "Elapsed search (FM/SAIS): $$((end-start)) sec, peak_rss_kb=$$peak_kb, query=$(OSATIVA_BENCH_QUERY)"

## time-search-osativa-all-suffixarray: measure search time on all Oryza sativa suffix-array index
time-search-osativa-all-suffixarray: $(OSATIVA_ALL_SA_INDEXFILE)
	@tmp=$$(mktemp); \
	start=$$(date +%s); \
	/usr/bin/time -f '%M' -o $$tmp ./$(BINARY) search --limit 10 $(OSATIVA_ALL_SA_INDEXFILE) "$(OSATIVA_BENCH_QUERY)" > /dev/null; \
	end=$$(date +%s); \
	peak_kb=$$(cat $$tmp); \
	rm -f $$tmp; \
	echo "Elapsed search (Stdlib/SuffixArray): $$((end-start)) sec, peak_rss_kb=$$peak_kb, query=$(OSATIVA_BENCH_QUERY)"

## bench-osativa-all: run build/search time comparisons for all Oryza sativa chromosomes
bench-osativa-all: time-build-index-osativa-all-sais time-build-index-osativa-all-suffixarray time-search-osativa-all-fm time-search-osativa-all-suffixarray

## time-build-index-moby-dick-sais: measure build time/memory/index size for Moby Dick
time-build-index-moby-dick-sais: download-moby-dick $(BINARY)
	@tmp=$$(mktemp); \
	start=$$(date +%s); \
	/usr/bin/time -f '%M' -o $$tmp ./$(BINARY) build --algo sais $(DATADIR)/moby_dick.txt $(DATADIR)/moby_dick.idx; \
	end=$$(date +%s); \
	peak_kb=$$(cat $$tmp); \
	index_bytes=$$(wc -c < $(DATADIR)/moby_dick.idx); \
	rm -f $$tmp; \
	echo "Elapsed build (Moby/FM/SAIS): $$((end-start)) sec, peak_rss_kb=$$peak_kb, index_bytes=$$index_bytes"

## time-search-moby-dick-fm: measure search time/memory for Moby Dick FM-index
time-search-moby-dick-fm: time-build-index-moby-dick-sais $(BINARY)
	@tmp=$$(mktemp); \
	start=$$(date +%s); \
	/usr/bin/time -f '%M' -o $$tmp ./$(BINARY) search --limit 10 $(DATADIR)/moby_dick.idx "$(MOBY_BENCH_QUERY)" > /dev/null; \
	end=$$(date +%s); \
	peak_kb=$$(cat $$tmp); \
	rm -f $$tmp; \
	echo "Elapsed search (Moby/FM/SAIS): $$((end-start)) sec, peak_rss_kb=$$peak_kb, query=$(MOBY_BENCH_QUERY)"

## bench-moby-dick: run build/search metrics for Moby Dick
bench-moby-dick: time-build-index-moby-dick-sais time-search-moby-dick-fm

## time-build-index-kenshin-sais: measure build time/memory/index size for Kenshin
time-build-index-kenshin-sais: download-kenshin $(BINARY)
	@tmp=$$(mktemp); \
	start=$$(date +%s); \
	/usr/bin/time -f '%M' -o $$tmp ./$(BINARY) build --algo sais $(DATADIR)/kenshin.txt $(DATADIR)/kenshin.idx; \
	end=$$(date +%s); \
	peak_kb=$$(cat $$tmp); \
	index_bytes=$$(wc -c < $(DATADIR)/kenshin.idx); \
	rm -f $$tmp; \
	echo "Elapsed build (Kenshin/FM/SAIS): $$((end-start)) sec, peak_rss_kb=$$peak_kb, index_bytes=$$index_bytes"

## time-search-kenshin-fm: measure search time/memory for Kenshin FM-index
time-search-kenshin-fm: time-build-index-kenshin-sais $(BINARY)
	@tmp=$$(mktemp); \
	start=$$(date +%s); \
	/usr/bin/time -f '%M' -o $$tmp ./$(BINARY) search --limit 10 $(DATADIR)/kenshin.idx "$(KENSHIN_BENCH_QUERY)" > /dev/null; \
	end=$$(date +%s); \
	peak_kb=$$(cat $$tmp); \
	rm -f $$tmp; \
	echo "Elapsed search (Kenshin/FM/SAIS): $$((end-start)) sec, peak_rss_kb=$$peak_kb, query=$(KENSHIN_BENCH_QUERY)"

## bench-kenshin: run build/search metrics for Kenshin
bench-kenshin: time-build-index-kenshin-sais time-search-kenshin-fm

## time-build-index-git-sais: measure build time/memory/index size for Git source index
time-build-index-git-sais: download-git $(BINARY)
	@tmp=$$(mktemp); \
	start=$$(date +%s); \
	find $(GIT_SRC_DIR) -type f \( -name "*.c" -o -name "*.h" \) | sort | /usr/bin/time -f '%M' -o $$tmp xargs ./$(BINARY) build-multi $(GIT_INDEXFILE); \
	end=$$(date +%s); \
	peak_kb=$$(cat $$tmp); \
	index_bytes=$$(wc -c < $(GIT_INDEXFILE)); \
	rm -f $$tmp; \
	echo "Elapsed build (Git/FM/SAIS): $$((end-start)) sec, peak_rss_kb=$$peak_kb, index_bytes=$$index_bytes"

## time-search-git-fm: measure search time/memory for Git FM-index
time-search-git-fm: time-build-index-git-sais $(BINARY)
	@tmp=$$(mktemp); \
	start=$$(date +%s); \
	/usr/bin/time -f '%M' -o $$tmp ./$(BINARY) search --limit 10 $(GIT_INDEXFILE) "$(GIT_BENCH_QUERY)" > /dev/null; \
	end=$$(date +%s); \
	peak_kb=$$(cat $$tmp); \
	rm -f $$tmp; \
	echo "Elapsed search (Git/FM/SAIS): $$((end-start)) sec, peak_rss_kb=$$peak_kb, query=$(GIT_BENCH_QUERY)"

## bench-git: run build/search metrics for Git source
bench-git: time-build-index-git-sais time-search-git-fm

## time-build-index-ecoli-sais: measure build time/memory/index size for E. coli
time-build-index-ecoli-sais: prepare-ecoli $(BINARY)
	@tmp=$$(mktemp); \
	start=$$(date +%s); \
	/usr/bin/time -f '%M' -o $$tmp ./$(BINARY) build --algo sais $(DATADIR)/ecoli.txt $(ECOLI_INDEXFILE); \
	end=$$(date +%s); \
	peak_kb=$$(cat $$tmp); \
	index_bytes=$$(wc -c < $(ECOLI_INDEXFILE)); \
	rm -f $$tmp; \
	echo "Elapsed build (Ecoli/FM/SAIS): $$((end-start)) sec, peak_rss_kb=$$peak_kb, index_bytes=$$index_bytes"

## time-search-ecoli-fm: measure search time/memory for E. coli FM-index
time-search-ecoli-fm: time-build-index-ecoli-sais $(BINARY)
	@tmp=$$(mktemp); \
	start=$$(date +%s); \
	/usr/bin/time -f '%M' -o $$tmp ./$(BINARY) search --limit 10 $(ECOLI_INDEXFILE) "$(ECOLI_BENCH_QUERY)" > /dev/null; \
	end=$$(date +%s); \
	peak_kb=$$(cat $$tmp); \
	rm -f $$tmp; \
	echo "Elapsed search (Ecoli/FM/SAIS): $$((end-start)) sec, peak_rss_kb=$$peak_kb, query=$(ECOLI_BENCH_QUERY)"

## bench-ecoli: run build/search metrics for E. coli
bench-ecoli: time-build-index-ecoli-sais time-search-ecoli-fm

## time-build-index-osativa-chr1-sais: measure build time/memory/index size for Oryza sativa chr1
time-build-index-osativa-chr1-sais: prepare-osativa-chr1 $(BINARY)
	@tmp=$$(mktemp); \
	start=$$(date +%s); \
	/usr/bin/time -f '%M' -o $$tmp ./$(BINARY) build --algo sais $(DATADIR)/osativa_chr1.txt $(OSATIVA_CHR1_INDEXFILE); \
	end=$$(date +%s); \
	peak_kb=$$(cat $$tmp); \
	index_bytes=$$(wc -c < $(OSATIVA_CHR1_INDEXFILE)); \
	rm -f $$tmp; \
	echo "Elapsed build (OsativaChr1/FM/SAIS): $$((end-start)) sec, peak_rss_kb=$$peak_kb, index_bytes=$$index_bytes"

## time-search-osativa-chr1-fm: measure search time/memory for Oryza sativa chr1 FM-index
time-search-osativa-chr1-fm: $(OSATIVA_CHR1_INDEXFILE) $(BINARY)
	@tmp=$$(mktemp); \
	start=$$(date +%s); \
	/usr/bin/time -f '%M' -o $$tmp ./$(BINARY) search --limit 10 $(OSATIVA_CHR1_INDEXFILE) "$(OSATIVA_CHR1_BENCH_QUERY)" > /dev/null; \
	end=$$(date +%s); \
	peak_kb=$$(cat $$tmp); \
	rm -f $$tmp; \
	echo "Elapsed search (OsativaChr1/FM/SAIS): $$((end-start)) sec, peak_rss_kb=$$peak_kb, query=$(OSATIVA_CHR1_BENCH_QUERY)"

## bench-osativa-chr1: run build/search metrics for Oryza sativa chr1
bench-osativa-chr1: time-build-index-osativa-chr1-sais time-search-osativa-chr1-fm

## time-build-index-amazon-small-sais: measure build time/memory/index size for Amazon small
time-build-index-amazon-small-sais: download-amazon-small prepare-amazon-small $(BINARY)
	@tmp=$$(mktemp); \
	start=$$(date +%s); \
	/usr/bin/time -f '%M' -o $$tmp ./$(BINARY) build --algo sais $(AMAZON_SMALL_TEXT) $(DATADIR)/amazon_small.idx; \
	end=$$(date +%s); \
	peak_kb=$$(cat $$tmp); \
	index_bytes=$$(wc -c < $(DATADIR)/amazon_small.idx); \
	rm -f $$tmp; \
	echo "Elapsed build (AmazonSmall/FM/SAIS): $$((end-start)) sec, peak_rss_kb=$$peak_kb, index_bytes=$$index_bytes"

## time-search-amazon-small-fm: measure search time/memory for Amazon small FM-index
time-search-amazon-small-fm: time-build-index-amazon-small-sais $(BINARY)
	@tmp=$$(mktemp); \
	start=$$(date +%s); \
	/usr/bin/time -f '%M' -o $$tmp ./$(BINARY) search --limit 10 $(DATADIR)/amazon_small.idx "$(AMAZON_BENCH_QUERY)" > /dev/null; \
	end=$$(date +%s); \
	peak_kb=$$(cat $$tmp); \
	rm -f $$tmp; \
	echo "Elapsed search (AmazonSmall/FM/SAIS): $$((end-start)) sec, peak_rss_kb=$$peak_kb, query=$(AMAZON_BENCH_QUERY)"

## bench-amazon-small: run build/search metrics for Amazon small
bench-amazon-small: time-build-index-amazon-small-sais time-search-amazon-small-fm

## time-build-index-amazon-medium-sais: measure build time/memory/index size for Amazon medium
time-build-index-amazon-medium-sais: download-amazon-medium prepare-amazon-medium $(BINARY)
	@tmp=$$(mktemp); \
	start=$$(date +%s); \
	/usr/bin/time -f '%M' -o $$tmp ./$(BINARY) build --algo sais $(AMAZON_MEDIUM_TEXT) $(DATADIR)/amazon_medium.idx; \
	end=$$(date +%s); \
	peak_kb=$$(cat $$tmp); \
	index_bytes=$$(wc -c < $(DATADIR)/amazon_medium.idx); \
	rm -f $$tmp; \
	echo "Elapsed build (AmazonMedium/FM/SAIS): $$((end-start)) sec, peak_rss_kb=$$peak_kb, index_bytes=$$index_bytes"

## time-search-amazon-medium-fm: measure search time/memory for Amazon medium FM-index
time-search-amazon-medium-fm: time-build-index-amazon-medium-sais $(BINARY)
	@tmp=$$(mktemp); \
	start=$$(date +%s); \
	/usr/bin/time -f '%M' -o $$tmp ./$(BINARY) search --limit 10 $(DATADIR)/amazon_medium.idx "$(AMAZON_BENCH_QUERY)" > /dev/null; \
	end=$$(date +%s); \
	peak_kb=$$(cat $$tmp); \
	rm -f $$tmp; \
	echo "Elapsed search (AmazonMedium/FM/SAIS): $$((end-start)) sec, peak_rss_kb=$$peak_kb, query=$(AMAZON_BENCH_QUERY)"

## bench-amazon-medium: run build/search metrics for Amazon medium
bench-amazon-medium: time-build-index-amazon-medium-sais time-search-amazon-medium-fm

## time-build-index-amazon-large-sais: measure build time/memory/index size for Amazon large
time-build-index-amazon-large-sais: download-amazon-large prepare-amazon-large $(BINARY)
	@tmp=$$(mktemp); \
	start=$$(date +%s); \
	/usr/bin/time -f '%M' -o $$tmp ./$(BINARY) build --algo sais $(AMAZON_LARGE_TEXT) $(DATADIR)/amazon_large.idx; \
	end=$$(date +%s); \
	peak_kb=$$(cat $$tmp); \
	index_bytes=$$(wc -c < $(DATADIR)/amazon_large.idx); \
	rm -f $$tmp; \
	echo "Elapsed build (AmazonLarge/FM/SAIS): $$((end-start)) sec, peak_rss_kb=$$peak_kb, index_bytes=$$index_bytes"

## time-search-amazon-large-fm: measure search time/memory for Amazon large FM-index
time-search-amazon-large-fm: time-build-index-amazon-large-sais $(BINARY)
	@tmp=$$(mktemp); \
	start=$$(date +%s); \
	/usr/bin/time -f '%M' -o $$tmp ./$(BINARY) search --limit 10 $(DATADIR)/amazon_large.idx "$(AMAZON_BENCH_QUERY)" > /dev/null; \
	end=$$(date +%s); \
	peak_kb=$$(cat $$tmp); \
	rm -f $$tmp; \
	echo "Elapsed search (AmazonLarge/FM/SAIS): $$((end-start)) sec, peak_rss_kb=$$peak_kb, query=$(AMAZON_BENCH_QUERY)"

## bench-amazon-large: run build/search metrics for Amazon large
bench-amazon-large: time-build-index-amazon-large-sais time-search-amazon-large-fm

## time-build-index-fake-log-apache-common-sais: measure build time/memory/index size for fake Apache common log
time-build-index-fake-log-apache-common-sais: generate-fake-log-apache-common $(BINARY)
	@tmp=$$(mktemp); \
	start=$$(date +%s); \
	/usr/bin/time -f '%M' -o $$tmp ./$(BINARY) build --algo sais $(DATADIR)/fake-logs/flog_apache_common.log $(DATADIR)/fake_logs_apache_common.idx; \
	end=$$(date +%s); \
	peak_kb=$$(cat $$tmp); \
	index_bytes=$$(wc -c < $(DATADIR)/fake_logs_apache_common.idx); \
	rm -f $$tmp; \
	echo "Elapsed build (FakeLogApacheCommon/FM/SAIS): $$((end-start)) sec, peak_rss_kb=$$peak_kb, index_bytes=$$index_bytes"

## time-search-fake-log-apache-common-fm: measure search time/memory for fake Apache common log FM-index
time-search-fake-log-apache-common-fm: time-build-index-fake-log-apache-common-sais $(BINARY)
	@tmp=$$(mktemp); \
	start=$$(date +%s); \
	/usr/bin/time -f '%M' -o $$tmp ./$(BINARY) search --limit 10 $(DATADIR)/fake_logs_apache_common.idx "$(FAKE_LOG_APACHE_COMMON_BENCH_QUERY)" > /dev/null; \
	end=$$(date +%s); \
	peak_kb=$$(cat $$tmp); \
	rm -f $$tmp; \
	echo "Elapsed search (FakeLogApacheCommon/FM/SAIS): $$((end-start)) sec, peak_rss_kb=$$peak_kb, query=$(FAKE_LOG_APACHE_COMMON_BENCH_QUERY)"

## bench-fake-log-apache-common: run build/search metrics for fake Apache common log
bench-fake-log-apache-common: time-build-index-fake-log-apache-common-sais time-search-fake-log-apache-common-fm

## time-build-index-fake-log-apache-error-sais: measure build time/memory/index size for fake Apache error log
time-build-index-fake-log-apache-error-sais: generate-fake-log-apache-error $(BINARY)
	@tmp=$$(mktemp); \
	start=$$(date +%s); \
	/usr/bin/time -f '%M' -o $$tmp ./$(BINARY) build --algo sais $(DATADIR)/fake-logs/flog_apache_error.log $(DATADIR)/fake_logs_apache_error.idx; \
	end=$$(date +%s); \
	peak_kb=$$(cat $$tmp); \
	index_bytes=$$(wc -c < $(DATADIR)/fake_logs_apache_error.idx); \
	rm -f $$tmp; \
	echo "Elapsed build (FakeLogApacheError/FM/SAIS): $$((end-start)) sec, peak_rss_kb=$$peak_kb, index_bytes=$$index_bytes"

## time-search-fake-log-apache-error-fm: measure search time/memory for fake Apache error log FM-index
time-search-fake-log-apache-error-fm: time-build-index-fake-log-apache-error-sais $(BINARY)
	@tmp=$$(mktemp); \
	start=$$(date +%s); \
	/usr/bin/time -f '%M' -o $$tmp ./$(BINARY) search --limit 10 $(DATADIR)/fake_logs_apache_error.idx "$(FAKE_LOG_APACHE_ERROR_BENCH_QUERY)" > /dev/null; \
	end=$$(date +%s); \
	peak_kb=$$(cat $$tmp); \
	rm -f $$tmp; \
	echo "Elapsed search (FakeLogApacheError/FM/SAIS): $$((end-start)) sec, peak_rss_kb=$$peak_kb, query=$(FAKE_LOG_APACHE_ERROR_BENCH_QUERY)"

## bench-fake-log-apache-error: run build/search metrics for fake Apache error log
bench-fake-log-apache-error: time-build-index-fake-log-apache-error-sais time-search-fake-log-apache-error-fm

## time-build-index-fake-log-syslog-sais: measure build time/memory/index size for fake syslog
time-build-index-fake-log-syslog-sais: generate-fake-log-syslog $(BINARY)
	@tmp=$$(mktemp); \
	start=$$(date +%s); \
	/usr/bin/time -f '%M' -o $$tmp ./$(BINARY) build --algo sais $(DATADIR)/fake-logs/flog_syslog.log $(DATADIR)/fake_logs_syslog.idx; \
	end=$$(date +%s); \
	peak_kb=$$(cat $$tmp); \
	index_bytes=$$(wc -c < $(DATADIR)/fake_logs_syslog.idx); \
	rm -f $$tmp; \
	echo "Elapsed build (FakeLogSyslog/FM/SAIS): $$((end-start)) sec, peak_rss_kb=$$peak_kb, index_bytes=$$index_bytes"

## time-search-fake-log-syslog-fm: measure search time/memory for fake syslog FM-index
time-search-fake-log-syslog-fm: time-build-index-fake-log-syslog-sais $(BINARY)
	@tmp=$$(mktemp); \
	start=$$(date +%s); \
	/usr/bin/time -f '%M' -o $$tmp ./$(BINARY) search --limit 10 $(DATADIR)/fake_logs_syslog.idx "$(FAKE_LOG_SYSLOG_BENCH_QUERY)" > /dev/null; \
	end=$$(date +%s); \
	peak_kb=$$(cat $$tmp); \
	rm -f $$tmp; \
	echo "Elapsed search (FakeLogSyslog/FM/SAIS): $$((end-start)) sec, peak_rss_kb=$$peak_kb, query=$(FAKE_LOG_SYSLOG_BENCH_QUERY)"

## bench-fake-log-syslog: run build/search metrics for fake syslog
bench-fake-log-syslog: time-build-index-fake-log-syslog-sais time-search-fake-log-syslog-fm

## time-build-index-fake-log-json-sais: measure build time/memory/index size for fake JSON log
time-build-index-fake-log-json-sais: generate-fake-log-json $(BINARY)
	@tmp=$$(mktemp); \
	start=$$(date +%s); \
	/usr/bin/time -f '%M' -o $$tmp ./$(BINARY) build --algo sais $(DATADIR)/fake-logs/mclogs_json.log $(DATADIR)/fake_logs_json.idx; \
	end=$$(date +%s); \
	peak_kb=$$(cat $$tmp); \
	index_bytes=$$(wc -c < $(DATADIR)/fake_logs_json.idx); \
	rm -f $$tmp; \
	echo "Elapsed build (FakeLogJSON/FM/SAIS): $$((end-start)) sec, peak_rss_kb=$$peak_kb, index_bytes=$$index_bytes"

## time-search-fake-log-json-fm: measure search time/memory for fake JSON FM-index
time-search-fake-log-json-fm: time-build-index-fake-log-json-sais $(BINARY)
	@tmp=$$(mktemp); \
	start=$$(date +%s); \
	/usr/bin/time -f '%M' -o $$tmp ./$(BINARY) search --limit 10 $(DATADIR)/fake_logs_json.idx "$(FAKE_LOG_JSON_BENCH_QUERY)" > /dev/null; \
	end=$$(date +%s); \
	peak_kb=$$(cat $$tmp); \
	rm -f $$tmp; \
	echo "Elapsed search (FakeLogJSON/FM/SAIS): $$((end-start)) sec, peak_rss_kb=$$peak_kb, query=$(FAKE_LOG_JSON_BENCH_QUERY)"

## bench-fake-log-json: run build/search metrics for fake JSON log
bench-fake-log-json: time-build-index-fake-log-json-sais time-search-fake-log-json-fm

## time-build-index-fake-log-logfmt-sais: measure build time/memory/index size for fake logfmt log
time-build-index-fake-log-logfmt-sais: generate-fake-log-logfmt $(BINARY)
	@tmp=$$(mktemp); \
	start=$$(date +%s); \
	/usr/bin/time -f '%M' -o $$tmp ./$(BINARY) build --algo sais $(DATADIR)/fake-logs/mclogs_logfmt.log $(DATADIR)/fake_logs_logfmt.idx; \
	end=$$(date +%s); \
	peak_kb=$$(cat $$tmp); \
	index_bytes=$$(wc -c < $(DATADIR)/fake_logs_logfmt.idx); \
	rm -f $$tmp; \
	echo "Elapsed build (FakeLogLogfmt/FM/SAIS): $$((end-start)) sec, peak_rss_kb=$$peak_kb, index_bytes=$$index_bytes"

## time-search-fake-log-logfmt-fm: measure search time/memory for fake logfmt FM-index
time-search-fake-log-logfmt-fm: time-build-index-fake-log-logfmt-sais $(BINARY)
	@tmp=$$(mktemp); \
	start=$$(date +%s); \
	/usr/bin/time -f '%M' -o $$tmp ./$(BINARY) search --limit 10 $(DATADIR)/fake_logs_logfmt.idx "$(FAKE_LOG_LOGFMT_BENCH_QUERY)" > /dev/null; \
	end=$$(date +%s); \
	peak_kb=$$(cat $$tmp); \
	rm -f $$tmp; \
	echo "Elapsed search (FakeLogLogfmt/FM/SAIS): $$((end-start)) sec, peak_rss_kb=$$peak_kb, query=$(FAKE_LOG_LOGFMT_BENCH_QUERY)"

## bench-fake-log-logfmt: run build/search metrics for fake logfmt log
bench-fake-log-logfmt: time-build-index-fake-log-logfmt-sais time-search-fake-log-logfmt-fm

## bench-fake-logs: run build/search metrics for all fake log variants
bench-fake-logs: bench-fake-log-apache-common bench-fake-log-apache-error bench-fake-log-syslog bench-fake-log-json bench-fake-log-logfmt

## bench-all-datasets-local: run build/search metrics for datasets without external tool/account dependencies
bench-all-datasets-local: bench-moby-dick bench-kenshin bench-git bench-ecoli bench-osativa-chr1 bench-osativa-all

## bench-all-datasets-external: run build/search metrics for datasets with external dependencies (Kaggle/flog/mclogs)
bench-all-datasets-external: bench-amazon-small bench-amazon-medium bench-amazon-large bench-fake-logs

## bench-all-datasets: run build/search metrics for all datasets (time/memory/index size)
bench-all-datasets: bench-all-datasets-local bench-all-datasets-external

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

## search-demo-osativa-all: run a sample search on the all-chromosome Oryza sativa index
search-demo-osativa-all: $(OSATIVA_ALL_INDEXFILE)
	./$(BINARY) search --limit 10 $(OSATIVA_ALL_INDEXFILE) "ATGGCG"

help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
