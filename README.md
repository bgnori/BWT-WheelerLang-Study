# BWT-WheelerLang-Study

全文検索アルゴリズムの理解と評価：FM-index（Wheeler グラフ）の実装と Go 標準ライブラリ `index/suffixarray` を統一インターフェースで選択・比較。

## 概要

このプロジェクトは以下を実装します：

- **FM-index（Wheeler グラフ）**：テキストの BWT（Burrows-Wheeler 変換）と SA（接尾辞配列）を組み合わせた全文検索インデックス。ノードの Wheeler 順序を満たすグラフとして表現されます。
- **ビットベクトルインデックス**：ランク演算（Rank1/Rank0）を O(1) でサポートする簡潔データ構造で、FM-index の Occ 配列を実現します。
- **星なし正規表現検索**：Kleene スターを含まない正規表現（星なし言語）によるパターン検索。制限を超えるクエリにはエラーメッセージを返します。
- **インデックスの永続化**：インデックスをバイナリファイルに保存・読み込みする機能。
- **標準ライブラリの Suffix Array**：`--algo suffixarray` オプションで Go の `index/suffixarray` バックエンドを選択できます。FM-index と同一の CLI インターフェースで使用でき、リテラル文字列検索に対応します（正規表現クエリは非対応）。

## ライブラリとして使う

外部プロジェクトからは、トップレベルの公開パッケージを利用できます。

```go
import bwtsearch "github.com/bgnori/bwt-wheelerlang-study"
```

主なAPI：

- `bwtsearch.Build`, `bwtsearch.BuildWithAlgorithm`, `bwtsearch.BuildWithOptions`, `bwtsearch.BuildFromFiles`
- `bwtsearch.Load`, `bwtsearch.ReadFrom`
- `(*bwtsearch.Index).Save`, `(*bwtsearch.Index).WriteTo`, `(*bwtsearch.Index).Append`, `(*bwtsearch.Index).Count`, `(*bwtsearch.Index).Locate`
- `bwtsearch.BuildStdlib`, `bwtsearch.BuildStdlibFromFiles`
- `(*bwtsearch.StdlibIndex).Save`, `(*bwtsearch.StdlibIndex).WriteTo`, `(*bwtsearch.StdlibIndex).Count`, `(*bwtsearch.StdlibIndex).Locate`
- `bwtsearch.LoadStdlib`, `bwtsearch.ReadStdlibFrom`
- `bwtsearch.Check`, `bwtsearch.Search`（星なし正規表現検索）

- エラー型: `bwtsearch.ViolationError`（星なし制約違反）、`bwtsearch.UnsupportedError`（非対応構文）

---

## 環境構築

### 開発環境と実行環境の使い分け

- **Dev Container (`.devcontainer/devcontainer.json`)**: VS Code での実装・テスト・デバッグ用の開発環境。
- **Docker (`Dockerfile`, `docker-compose.yml`)**: CLI の実行・配布確認用のコンテナ環境。

役割を分けることで、開発効率（Dev Container）と実行再現性（Docker）の両方を維持します。

### Docker を使う場合（推奨）

```bash
# イメージをビルド
docker compose build

# テストデータをダウンロード（コミット禁止）
docker compose run download

# FM-index を構築
docker compose run bwtsearch build /data/moby_dick.txt /data/moby_dick.idx

# 検索
docker compose run bwtsearch search /data/moby_dick.idx "white whale"

# 上杉謙信テキストをダウンロード・変換（Shift-JIS → UTF-8 を自動処理）
docker compose run download-kenshin

# 上杉謙信 FM-index を構築
docker compose run bwtsearch build /data/kenshin.txt /data/kenshin.idx

# 上杉謙信を検索
docker compose run bwtsearch search /data/kenshin.idx "上杉謙信"
```

### ローカルで Go を使う場合

Go 1.21 以上が必要です。

```bash
# ビルド
make build

# テスト
make test

# テストデータのダウンロード（コミット禁止）
make download-testdata

# FM-index の構築
make build-index

# サンプル検索
make search-demo
```

---

## コマンドリファレンス

### `build` — インデックス構築

```
bwtsearch build [--algo doubling|sais|suffixarray] [--occ bitvectors|wavelet] <input-file> <index-file>
```

テキストファイルからインデックスを構築し、バイナリ形式でファイルに保存します。

- `--algo`: 接尾辞配列の構築アルゴリズム（既定値 `doubling`）
  - `doubling`: 前置倍加法（Manber-Myers）。シンプルで安定。FM-index を構築します。
  - `sais`: SA-IS アルゴリズム。大規模テキストで高速。FM-index を構築します。
  - `suffixarray`: Go 標準ライブラリ `index/suffixarray` を使用。リテラル検索専用。正規表現クエリは非対応。
- `--occ`: Occ 配列の実装（既定値 `bitvectors`）
  - `bitvectors`: 文字ごとの簡潔ビットベクトル。アルファベットが小さい場合に有利。
  - `wavelet`: ウェーブレット木。アルファベットが大きい（バイト値の種類が多い）テキストで有利。

**例：**

```bash
# デフォルト設定（前置倍加 + ビットベクトル）
bwtsearch build data/moby_dick.txt data/moby_dick.idx

# SA-IS アルゴリズムを使用
bwtsearch build --algo sais data/moby_dick.txt data/moby_dick.idx

# SA-IS + ウェーブレット木を使用
bwtsearch build --algo sais --occ wavelet data/moby_dick.txt data/moby_dick.idx

# Go 標準ライブラリの Suffix Array を使用（リテラル検索専用）
bwtsearch build --algo suffixarray data/moby_dick.txt data/moby_dick.saidx
```

### `build-multi` — 複数ファイルからインデックス構築

```
bwtsearch build-multi [--algo doubling|sais|suffixarray] [--occ bitvectors|wavelet] <index-file> <file1> [file2 ...]
```

複数のテキストファイルを結合して FM-index を構築します。ファイルの区切りには改行（`\n`）が使われます。

- `--algo`/`--occ` オプションは `build` コマンドと同じです。

**例：**

```bash
bwtsearch build-multi --algo sais data/combined.idx data/file1.txt data/file2.txt
```

### `info` — インデックス情報表示

```
bwtsearch info <index-file>
```

テキスト長・SA サイズ・アルファベットサイズを表示します。

### `graph` — Wheeler グラフを Mermaid で出力

```
bwtsearch graph [flags] <index-file>
```

FM-index が内部で表現している Wheeler グラフを Mermaid 形式で出力します。

- `--max-nodes N`: 描画するノード上限（Wheeler 順序の先頭から）。`0` で全ノード。
- `--markdown`: `true` のとき ```mermaid フェンス付きで出力（既定値 `true`）。

**例：**

```bash
bwtsearch graph data/moby_dick.idx --max-nodes 12 > docs/wheeler_graph_moby.md
```

`docs/wheeler_graph_examples.md` には小規模テキストの具体例を用意しています。

### `browse` — インタラクティブブラウズ

```
bwtsearch browse <index-file> [--show N] [--context N]
```

端末でインタラクティブに接尾辞配列を探索します。文字を入力するたびに検索結果が絞り込まれます。Backspace で削除、空 Enter で終了。

### `search` — 検索

```
bwtsearch search [flags] <index-file> <pattern>
  --limit N      最大結果件数 (0 = 無制限, default 20)
  --context N    各マッチの前後表示文字数 (default 80)
  --positions    テキスト位置のみ表示
```

インデックスのバックエンドに応じて自動的に検索方式を選択します。

- **FM-index**（`--algo doubling` または `--algo sais` で構築）：星なし正規表現によるパターン検索。
- **Suffix Array**（`--algo suffixarray` で構築）：リテラル文字列検索のみ。正規表現クエリは非対応。

**FM-index でサポートする演算子：**

| 構文          | 意味                       | 星なし言語内？ |
|---------------|---------------------------|----------------|
| `hello`       | リテラル文字列             | ✓              |
| `.`           | 任意の 1 文字（改行除く）  | ✓              |
| `[abc]`       | 文字クラス                 | ✓              |
| `a\|b`        | 選択（和集合）             | ✓              |
| `(group)`     | グループ化                 | ✓              |
| `a?`          | 省略可能（= a\|ε）         | ✓              |
| `a{n,m}`      | 有界繰り返し               | ✓              |
| `a*`          | Kleene スター              | **✗ 拒否**     |
| `a+`          | 1 回以上                   | **✗ 拒否**     |
| `a{n,}`       | 無界繰り返し               | **✗ 拒否**     |

星なし言語の範囲を超えるクエリを受け取った場合、どの部分が制限を超えているかを含むエラーメッセージを返します。

**例：**

```bash
# リテラル検索（FM-index・Suffix Array どちらでも可）
bwtsearch search --limit 20 moby.idx "white whale"

# 選択（FM-index のみ）
bwtsearch search --limit 10 moby.idx "whale|ship"

# 文字クラス（FM-index のみ）
bwtsearch search moby.idx "[Ww]hale"

# 星なし言語を超えるクエリ（FM-index でエラーになる例）
bwtsearch search moby.idx "wha.*"
# → Pattern rejected: star-free violation: ".*" uses Kleene star (*)...

# Suffix Array インデックスに対するリテラル検索
bwtsearch search --limit 20 moby.saidx "white whale"
```

### `web` — 検索を試せる簡易 Web アプリ

```
bwtsearch web [--index FILE] [--addr ADDR] [--limit N] [--context N] [--min-chars N]
```

インデックスを読み込んでローカル HTTP サーバーを起動し、ブラウザ上で検索を試せます。FM-index バックエンドの場合は星なし正規表現検索が可能です。

- `--index FILE`: 読み込むインデックス（既定値 `data/moby_dick.idx`）
- `--addr ADDR`: 待ち受けアドレス（既定値 `:8080`）
- `--limit N`: 初期の最大表示件数（既定値 `20`）
- `--context N`: 初期の前後コンテキスト文字数（既定値 `80`）
- `--min-chars N`: 入力中の自動検索を開始する最小文字数（既定値 `4`）

**例：**

```bash
bwtsearch web --index data/moby_dick.idx --addr :8080
# ブラウザで http://localhost:8080 を開く
```

---

## テストデータ

### Moby Dick（英語サンプル）

[Project Gutenberg #2701 (Moby Dick)](https://www.gutenberg.org/ebooks/2701) を使用します。

```bash
./scripts/download_testdata.sh
# または
make download-testdata
```

### 上杉謙信（非ASCII・日本語サンプル）

[青空文庫 図書カード No.56461「上杉謙信」](https://www.aozora.gr.jp/cards/001562/card56461.html) を使用します。

青空文庫のテキストは **Shift-JIS (CP932)** でエンコードされています。ダウンロードスクリプトは Zip を展開後、`iconv`（または Python）を使って **UTF-8** に変換してから `data/kenshin.txt` に保存します。FM-index はバイト列として動作するため、UTF-8 変換後のテキストをそのまま索引化できます。

```bash
./scripts/download_kenshin.sh
# または
make download-kenshin
```

#### ローカル（Go）での手順

```bash
# 1. テキストをダウンロード・変換
make download-kenshin

# 2. FM-index を構築
make build-index-kenshin

# 3. サンプル検索（「上杉謙信」をリテラル検索）
make search-demo-kenshin
```

直接 CLI で操作する場合:

```bash
# インデックス構築
./bwtsearch build data/kenshin.txt data/kenshin.idx

# 検索（UTF-8 パターンをそのまま指定）
./bwtsearch search data/kenshin.idx "上杉謙信" --limit 5

# インデックス情報
./bwtsearch info data/kenshin.idx
```

#### Docker での手順

```bash
# イメージをビルド
docker compose build

# テキストをダウンロード・変換
docker compose run download-kenshin

# FM-index を構築
docker compose run bwtsearch build /data/kenshin.txt /data/kenshin.idx

# 検索
docker compose run bwtsearch search /data/kenshin.idx "上杉謙信"
```

#### 非ASCII文字（UTF-8）処理の注意点

- **エンコーディング**: 青空文庫テキストは Shift-JIS (CP932) です。スクリプトが自動で UTF-8 に変換します。
- **インデックスはバイト単位**: FM-index の後方検索はバイト列上で動作します。UTF-8 の日本語文字は 3 バイトなので、検索パターンも UTF-8 文字列をそのまま渡せます。
- **コンテキスト表示の境界補正**: `ContextAround` は表示スニペットの前後を UTF-8 のルーン境界に合わせて調整するため、文字が途中で切れて文字化けすることはありません。
- **青空文庫ルビ記法**: テキストには `《》` などのルビ記法が残っています。検索パターンに含めることも可能です。
- **ゼロバイト**: FM-index はゼロバイト（0x00）を番兵として使用します。UTF-8 テキストにゼロバイトが含まれる場合は索引化前に除去が必要ですが、通常の青空文庫テキストには含まれません。

> **注意：** テキストデータおよびインデックスファイル（`data/*.txt`, `data/*.idx`）は `.gitignore` に登録されており、リポジトリにはコミットしないでください。

### E. coli K-12 MG1655 / イネ第1染色体（ゲノムデータサンプル）

[NCBI RefSeq NC_000913.3](https://www.ncbi.nlm.nih.gov/nuccore/NC_000913.3) — 大腸菌（*Escherichia coli*）K-12 株 MG1655 の完全ゲノム（約 4.6 Mbp）を使用します。
データソースは NCBI FTP（Assembly: GCF_000005845.2, ASM584v2）です。

ゲノムデータは **FASTA 形式**（`.fna`）で配布されています。FM-index に取り込む前に、FASTA ヘッダ行（`>` で始まる行）を除去し、塩基配列を大文字化・結合した平文テキスト（`.txt`）に変換する必要があります。
この前処理は `prepare_ecoli.sh` を使い回しでき、入力/出力ファイル名を切り替えるだけで他の FASTA にも適用できます。

#### ローカル（Go）での手順

```bash
# 1. FASTA ファイルをダウンロード → data/ecoli.fna
make download-ecoli

# 2. FASTA を平文 DNA テキストに変換 → data/ecoli.txt
make prepare-ecoli

# 3. FM-index を構築（大規模入力のため SA-IS を推奨）
make build-index-ecoli

# 4. サンプル検索
make search-demo-ecoli
```

直接スクリプト・CLI で操作する場合:

```bash
# FASTA ダウンロード
./scripts/download_ecoli.sh

# FASTA → 平文 DNA 変換
./scripts/prepare_ecoli.sh

# FM-index 構築
./bwtsearch build --algo sais data/ecoli.txt data/ecoli.idx

# 検索（例：開始コドン周辺の典型的なパターン）
./bwtsearch search data/ecoli.idx "ATGAAACGC" --limit 10

# インデックス情報
./bwtsearch info data/ecoli.idx
```

### Oryza sativa（イネ）第1染色体

[NCBI GenBank Assembly GCA_001433935.1 (IRGSP-1.0)](https://www.ncbi.nlm.nih.gov/datasets/genome/GCA_001433935.1/) の第1染色体 FASTA（約 43.3MB〜45MB）を使用できます。

```bash
# 1. FASTA ファイルをダウンロード → data/osativa_chr1.fna
make download-osativa-chr1

# 2. FASTA を平文 DNA テキストに変換（既存スクリプトを再利用）→ data/osativa_chr1.txt
make prepare-osativa-chr1

# 3. FM-index を構築（大規模入力のため SA-IS を推奨）
make build-index-osativa-chr1

# 4. サンプル検索
make search-demo-osativa-chr1
```

直接スクリプト・CLI で操作する場合:

```bash
./scripts/download_osativa_chr1.sh
./scripts/prepare_ecoli.sh ./data osativa_chr1.fna osativa_chr1.txt
./bwtsearch build --algo sais data/osativa_chr1.txt data/osativa_chr1.idx
./bwtsearch search data/osativa_chr1.idx "ATGGCG" --limit 10
./bwtsearch info data/osativa_chr1.idx
```

#### Docker での手順

```bash
# イメージをビルド
docker compose build

# FASTA ダウンロード＆変換（ecoli.fna と ecoli.txt を生成）
docker compose run download-ecoli

# FM-index を構築
docker compose run bwtsearch build --algo sais /data/ecoli.txt /data/ecoli.idx

# 検索
docker compose run bwtsearch search /data/ecoli.idx "ATGAAACGC"

# イネ第1染色体の FASTA ダウンロード＆変換（osativa_chr1.fna と osativa_chr1.txt を生成）
docker compose run download-osativa-chr1

# イネ第1染色体インデックスを構築
docker compose run bwtsearch build --algo sais /data/osativa_chr1.txt /data/osativa_chr1.idx
```

#### ゲノムデータ処理の注意点

- **FASTA 形式**: ヘッダ行（`>`）と塩基配列行が交互に並ぶテキスト形式です。`prepare_ecoli.sh` がヘッダを除去し、各レコードの配列を 1 行に結合して大文字化します。
- **文字セット**: E. coli ゲノムは A/C/G/T/N（不明塩基）のみで構成されます。すべて ASCII 1 バイトなので FM-index にそのまま取り込めます。
- **大規模テキスト**: 約 4.6 MB・460 万文字の連続バイト列です。インデックス構築には `--algo sais`（SA-IS アルゴリズム）を推奨します。
- **植物ゲノムの反復配列**: イネを含む植物ゲノムは反復配列が多く、BWT ラン分布や検索ヒット分布が偏りやすいため、インデックス生成時間・メモリ使用量・検索時の候補数に影響が出やすい点に注意してください。
- **ゼロバイト**: FM-index はゼロバイト（0x00）を番兵として使用します。通常の FASTA 塩基配列にゼロバイトは含まれません。
- **`.fna` ファイル**: ダウンロードされた生 FASTA ファイル（`data/ecoli.fna`）も `.gitignore` で除外されています。コミットしないでください。

> **注意：** FASTA ファイル（`data/*.fna`）・平文テキスト（`data/*.txt`）・インデックス（`data/*.idx`）はすべて `.gitignore` に登録されており、リポジトリにはコミットしないでください。

### Kaggle Amazon データセット（小・中・大）

Kaggle の以下 3 データセットを、テスト/ベンチマーク用コーパスとして扱えます（**データ本体はコミット禁止**）。

- **小**: Amazon Laptop Prices Dataset（既定: `ionaskel/laptop-prices`）
- **中**: Amazon Mobile Dataset（既定: `PromptCloudHQ/amazon-unlocked-mobile`）
- **大**: Amazon Product Dataset (100K+)（既定: `piyushjain16/amazon-product-dataset`）

#### 前提

- Kaggle CLI が必要です（`pip install kaggle`）。
- Kaggle API 認証（`~/.kaggle/kaggle.json` または `KAGGLE_USERNAME`/`KAGGLE_KEY`）を設定してください。
- データセット ID は必要に応じて環境変数で上書きできます。
  - `KAGGLE_DATASET_SMALL`
  - `KAGGLE_DATASET_MEDIUM`
  - `KAGGLE_DATASET_LARGE`

#### ローカル（Go）での手順

```bash
# 小
make download-amazon-small
make prepare-amazon-small
make build-index-amazon-small

# 中
make download-amazon-medium
make prepare-amazon-medium
make build-index-amazon-medium

# 大
make download-amazon-large
make prepare-amazon-large
make build-index-amazon-large
```

生データは `data/kaggle/<small|medium|large>/` に展開され、前処理後テキストは `data/amazon_<size>.txt` に出力されます。前処理では CSV/TSV の複数列（商品名・ブランド・説明など）を正規化して 1 行テキストにまとめ、FM-index 入力に使える形式へ変換します。

> **注意：** Kaggle の zip/csv/tsv などの生データと、生成される `.txt` / `.idx` は `.gitignore` で除外されます。リポジトリへコミットしないでください。

### Git LFS について

このリポジトリは **現時点では Git LFS を必須としていません**。大容量になりやすいデータは `data/` 配下に置き、`.gitignore` で管理します。

- `git: 'lfs' is not a git command` が表示される場合: 現在の開発コンテナに `git-lfs` が入っていない状態です。
- 通常開発（コード編集・テスト・比較実験）では、そのまま作業を継続できます。
- もし将来、大容量アセットを Git 管理下に置く必要が出た場合は、`git-lfs` を導入してから対象拡張子を明示的に追跡してください。

---

## アーキテクチャ

```
internal/
  bitvector/   ── 簡潔ビットベクトル（Rank1/Rank0）
  wavelet/     ── ウェーブレット木（O(log σ) Rank 演算）
  fmindex/     ── FM-index：BWT・SA・Occ・C 配列の構築・検索・永続化
  starfree/    ── 星なし正規表現の検証と FM-index 上の後方検索
cmd/bwtsearch/ ── CLI エントリポイント
scripts/       ── テストデータダウンロードスクリプト
data/          ── テストデータ置き場（.gitignore 対象）
```

### Wheeler グラフとの対応

FM-index は文字列に対する Wheeler グラフの圧縮表現です：

| Wheeler グラフ概念        | FM-index での実現             |
|--------------------------|-------------------------------|
| ノード                    | 接尾辞配列の各位置（F 列）    |
| エッジラベル              | BWT 文字（L 列）              |
| Wheeler 順序              | 接尾辞の辞書式順序            |
| ノードのラベル付け関数    | C 配列 + Occ ビットベクトル   |

後方検索（backward search）は Wheeler グラフ上の節点区間を 1 文字ずつ左に延長する操作に対応します。

---

## テスト実行

```bash
# 全テスト（race condition 検出付き）
make test

# 詳細出力
make test-verbose
```

---

## ライセンス

MIT — 詳細は [LICENSE](LICENSE) を参照。
