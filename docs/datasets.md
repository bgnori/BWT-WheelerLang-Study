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

## 5. Oryza sativa（イネ）

- ソース: [NCBI GenBank Assembly GCA_001433935.1](https://www.ncbi.nlm.nih.gov/datasets/genome/GCA_001433935.1/)
- `download-osativa` は Oryza sativa 全体のゲノム FASTA を `data/osativa.fna` として取得します。
- `prepare-osativa-chr1` は `data/osativa.fna` から指定に一致する FASTA レコードだけを抽出し、索引対象の `data/osativa_chr1.txt` を作成します。
- `prepare-osativa-all` は `data/osativa.fna` の全 FASTA レコード（全 chromosome）を結合し、索引対象の `data/osativa_all.txt` を作成します。
- 既定の抽出対象は chromosome 1（`AP014957.1`）です。別の FASTA レコードを対象にする場合は `OSATIVA_CHR1_SELECTOR` に awk 拡張正規表現を指定します。

大規模データセットとして使う場合は、次の 2 オプションを選択できます。

- chromosome 1 のみ: サイズを抑えて試行しやすい
- 全 chromosome: 最大規模で性能評価しやすい

```bash
# 1. 全体 FASTA を取得
make download-osativa

# 2-A. chromosome 1 を抽出して index 対象ファイルを作成
make prepare-osativa-chr1

# 3-A. chr1 抽出済みファイルを索引化して検索
make build-index-osativa-chr1
make search-demo-osativa-chr1

# 2-B. 全 chromosome を結合して index 対象ファイルを作成
make prepare-osativa-all

# 3-B. 全 chromosome 結合済みファイルを索引化して検索
make build-index-osativa-all
make search-demo-osativa-all

# 4-B. 全 chromosome で suffix array 比較
make suffixarray-demo-osativa-all

# 5-B. 全 chromosome で性能計測ベンチ（時間・メモリ・インデックスサイズ）
make bench-osativa-all

# クエリを変えて検索時間・メモリを比較する例
make time-search-osativa-all-fm OSATIVA_BENCH_QUERY='GTTACCTGCC'
make time-search-osativa-all-suffixarray OSATIVA_BENCH_QUERY='GTTACCTGCC'
```

`download-osativa-chr1` は互換エイリアスとして残していますが、取得されるファイルは chromosome 1 単体ではなく `data/osativa.fna` の全体 FASTA です。

```bash
make download-osativa-chr1
```

抽出対象を変更する例:

```bash
# chromosome 2 を data/osativa_chr1.txt に抽出する場合
rm -f data/osativa_chr1.txt
make prepare-osativa-chr1 OSATIVA_CHR1_SELECTOR='AP014958[.]1'
```

## 6. Kaggle Amazon データセット（小/中/大）

Kaggle 上の Amazon 関連データセットはコミュニティ投稿が中心です。small は `owm4096/laptop-prices` を既定値として使いますが、公式データソースではありません。必要に応じて、利用するデータセットを確認したうえで `KAGGLE_DATASET_SMALL` / `KAGGLE_DATASET_MEDIUM` / `KAGGLE_DATASET_LARGE` に明示してください。

前提:

- Kaggle CLI（`pip install kaggle`）
- `~/.kaggle/kaggle.json`、`~/.kaggle/access_token`、または `KAGGLE_USERNAME` / `KAGGLE_KEY`

例:

```bash
KAGGLE_DATASET_SMALL=owner/dataset-slug make download-amazon-small
KAGGLE_DATASET_MEDIUM=owner/dataset-slug make download-amazon-medium
```

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

## 8. 全データセット一括ベンチ（時間・メモリ・インデックスサイズ）

全データセットを一括で測定するには以下を実行します。

```bash
make bench-all-datasets
```

外部依存の有無で分けて実行する場合:

```bash
# 外部依存なし（Kaggle/flog/mclogs 不要）
make bench-all-datasets-local

# 外部依存あり（Kaggle + flog/mclogs が必要）
make bench-all-datasets-external
```

個別に実行したい場合:

```bash
make bench-moby-dick
make bench-kenshin
make bench-git
make bench-ecoli
make bench-osativa-chr1
make bench-osativa-all
make bench-amazon-small
make bench-amazon-medium
make bench-amazon-large
make bench-fake-logs
```

## Docker での利用

主要なデータ取得は `docker compose run <service>` でも実行できます（例: `download`, `download-kenshin`, `download-git`, `download-ecoli`, `download-osativa`）。
