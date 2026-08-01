# textindex

全文検索アルゴリズムの理解と評価のために、FM-index 系実装と Go 標準ライブラリ `index/suffixarray` を同一 CLI/API で扱えるようにしたリポジトリです。

## とりあえず試す（最短）

### Docker ですぐ試す（推奨）

```bash
docker compose build
docker compose run download
docker compose run textindex build /data/moby_dick.txt /data/moby_dick.idx
docker compose run textindex search /data/moby_dick.idx "white whale"
```

### ローカル Go ですぐ試す

```bash
make build
make download-moby-dick
make build-index-moby-dick
make search-demo-moby-dick
```

## 詳細ドキュメント

- CLI の全コマンド・オプション: [`docs/cli_reference.md`](docs/cli_reference.md)
- テストデータ準備（Moby Dick / 上杉謙信 / Git / ゲノム / Kaggle / Fake ログ）: [`docs/datasets.md`](docs/datasets.md)
- ライブラリ API: [`docs/library_api.md`](docs/library_api.md)
- 性能計測: [`docs/performance_report.md`](docs/performance_report.md)
- 比較・調査メモ: [`docs/library_comparison.md`](docs/library_comparison.md), [`docs/issues.md`](docs/issues.md), [`docs/review.md`](docs/review.md)
- Wheeler グラフ例: [`docs/wheeler_graph_examples.md`](docs/wheeler_graph_examples.md)

## 概要

実装している主なバックエンド:

- `--algo doubling` / `--algo sais`: FM-index
- `--algo suffixarray`: Go 標準ライブラリ Suffix Array（リテラル検索専用）
- `--algo bifmindex`: 双方向 FM-index（リテラル検索）

FM-index 系 (`doubling` / `sais` / `bifmindex`) では Occ 構造を選択できます:

- `--occ bitvectors`
- `--occ wavelet`
- `--occ waveletmatrix`
- `--occ rlbwt`（既定）
- `--occ rrr`
- `--occ eliasfano`
- `--occ poppy`（Interleaved RRR）
- `--occ dynamic`

物理配置は Occ 構造と直交に選択できます:

- `--storage memory`（既定）
- `--storage external`（現状は `--occ wavelet` で有効）
- `--disk-block-size BYTES`（`--storage external` 時のブロックサイズ、既定 4096）

## よく使うコマンド

```bash
# インデックス作成
textindex build [--algo doubling|sais|suffixarray|bifmindex] \
  [--occ bitvectors|wavelet|waveletmatrix|rlbwt|rrr|eliasfano|poppy|dynamic] \
  [--storage memory|external] [--disk-block-size BYTES] <input-file> <index-file>

# 複数ファイルまとめて作成
textindex build-multi [--algo doubling|sais|suffixarray|bifmindex] \
  [--occ bitvectors|wavelet|waveletmatrix|rlbwt|rrr|eliasfano|poppy|dynamic] \
  [--storage memory|external] [--disk-block-size BYTES] <index-file> <file1> [file2 ...]

# 検索
textindex search [--limit N] [--context N] [--positions] <index-file> <pattern>

# Web UI
textindex web [--index FILE] [--addr ADDR] [--limit N] [--context N] [--min-chars N]
```

> `search` のパターン解釈はバックエンド依存です。
>
> - FM-index（`doubling` / `sais`）: 星なし正規表現
> - Suffix Array / 双方向 FM-index: リテラル検索

## 開発

```bash
make test
make test-verbose
make lint
```

## ライセンス

MIT — [LICENSE](LICENSE)
