# Test Data Guide

このリポジトリのデータ取得・生成用コマンドをまとめます。生成物（`data/*.txt`, `data/*.idx`, `data/*.fna`, `data/fake-logs/*` など）は `.gitignore` 対象で、コミットしません。

## 1. Moby Dick（英語）

- ソース: [Project Gutenberg #2701](https://www.gutenberg.org/ebooks/2701)

```bash
make download-moby-dick
make build-index-moby-dick
make search-demo-moby-dick
```

## 2. 上杉謙信（日本語）

- ソース: [青空文庫 図書カード No.56461](https://www.aozora.gr.jp/cards/001562/card56461.html)
- `download_kenshin.sh` が Shift-JIS (CP932) から UTF-8 に変換します。

```bash
make download-kenshin
make build-index-kenshin
make search-demo-kenshin
```

Suffix Array 比較:

```bash
make suffixarray-demo-kenshin
```

## 3. Git ソースコード（C / H）

- ソース: [Git v2.45.0](https://github.com/git/git/releases/tag/v2.45.0)

```bash
make download-git
make build-index-git
make search-demo-git
make suffixarray-demo-git
```

## 4. E. coli ゲノム

- ソース: [NCBI RefSeq NC_000913.3](https://www.ncbi.nlm.nih.gov/nuccore/NC_000913.3)
- FASTA を平文 DNA へ変換後に索引化します。

```bash
make download-ecoli
make prepare-ecoli
make build-index-ecoli
make search-demo-ecoli
make suffixarray-demo-ecoli
```

## 5. Oryza sativa（イネ）第1染色体

- ソース: [NCBI GenBank Assembly GCA_001433935.1](https://www.ncbi.nlm.nih.gov/datasets/genome/GCA_001433935.1/)

```bash
make download-osativa-chr1
make prepare-osativa-chr1
make build-index-osativa-chr1
make search-demo-osativa-chr1
```

## 6. Kaggle Amazon データセット（小/中/大）

前提:

- Kaggle CLI（`pip install kaggle`）
- `~/.kaggle/kaggle.json` または `KAGGLE_USERNAME` / `KAGGLE_KEY`

```bash
# small
make download-amazon-small
make prepare-amazon-small
make build-index-amazon-small

# medium
make download-amazon-medium
make prepare-amazon-medium
make build-index-amazon-medium

# large
make download-amazon-large
make prepare-amazon-large
make build-index-amazon-large
```

必要に応じて `KAGGLE_DATASET_SMALL` / `KAGGLE_DATASET_MEDIUM` / `KAGGLE_DATASET_LARGE` でデータセット ID を上書きできます。

## 7. Fake ログ（flog / mclogs）

```bash
# 5種類をまとめて生成（既定 1M）
make generate-fake-logs

# サイズ指定
make generate-fake-logs FAKE_LOG_SIZE=10M

# 個別
make generate-fake-log-apache-common FAKE_LOG_SIZE=5M
make generate-fake-log-apache-error FAKE_LOG_SIZE=5M
make generate-fake-log-syslog FAKE_LOG_SIZE=5M
make generate-fake-log-json FAKE_LOG_SIZE=5M
make generate-fake-log-logfmt FAKE_LOG_SIZE=5M
```

`flog` / `mclogs` コマンドが必要です。

## Docker での利用

主要なデータ取得は `docker compose run <service>` でも実行できます（例: `download`, `download-kenshin`, `download-git`, `download-ecoli`, `download-osativa-chr1`）。
