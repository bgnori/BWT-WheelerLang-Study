# BWT-WheelerLang-Study

全文検索アルゴリズムの理解と評価：FM-index（Wheeler グラフ）の実装と Go 標準ライブラリ `index/suffixarray` との機能比較。

## 概要

このプロジェクトは以下を実装します：

- **FM-index（Wheeler グラフ）**：テキストの BWT（Burrows-Wheeler 変換）と SA（接尾辞配列）を組み合わせた全文検索インデックス。ノードの Wheeler 順序を満たすグラフとして表現されます。
- **ビットベクトルインデックス**：ランク演算（Rank1/Rank0）を O(1) でサポートする簡潔データ構造で、FM-index の Occ 配列を実現します。
- **星なし正規表現検索**：Kleene スターを含まない正規表現（星なし言語）によるパターン検索。制限を超えるクエリにはエラーメッセージを返します。
- **インデックスの永続化**：インデックスをバイナリファイルに保存・読み込みする機能。
- **標準ライブラリとの比較**：Go の `index/suffixarray` と件数・速度を比較する `compare` コマンド。

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

# FM-index vs 標準ライブラリの比較
make compare-demo
```

---

## コマンドリファレンス

### `build` — インデックス構築

```
bwtsearch build <input-file> <index-file>
```

テキストファイルから FM-index を構築し、バイナリ形式でファイルに保存します。

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

### `search` — 星なし正規表現検索

```
bwtsearch search [flags] <index-file> <pattern>
  --limit N      最大結果件数 (0 = 無制限, default 20)
  --context N    各マッチの前後表示文字数 (default 80)
  --positions    テキスト位置のみ表示
```

**サポートする演算子：**

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
# リテラル検索
bwtsearch search --limit 20 moby.idx "white whale"

# 選択
bwtsearch search --limit 10 moby.idx "whale|ship"

# 文字クラス
bwtsearch search moby.idx "[Ww]hale"

# 星なし言語を超えるクエリ（エラーになる例）
bwtsearch search moby.idx "wha.*"
# → Pattern rejected: star-free violation: ".*" uses Kleene star (*)...
```

### `compare` — 標準ライブラリとの比較

```
bwtsearch compare <input-file> <pattern> [--limit N]
```

同一テキストに対して FM-index と Go 標準の `index/suffixarray` を実行し、件数と処理時間を比較します。

### `web` — 検索を試せる簡易 Web アプリ

```
bwtsearch web [--index FILE] [--addr ADDR] [--limit N] [--context N]
```

インデックスを読み込んでローカル HTTP サーバーを起動し、ブラウザ上で星なし正規表現検索を試せます。

- `--index FILE`: 読み込むインデックス（既定値 `data/moby_dick.idx`）
- `--addr ADDR`: 待ち受けアドレス（既定値 `:8080`）
- `--limit N`: 初期の最大表示件数（既定値 `20`）
- `--context N`: 初期の前後コンテキスト文字数（既定値 `80`）

**例：**

```bash
bwtsearch web --index data/moby_dick.idx --addr :8080
# ブラウザで http://localhost:8080 を開く
```

---

## テストデータ

[Project Gutenberg #2701 (Moby Dick)](https://www.gutenberg.org/ebooks/2701) を使用します。

```bash
./scripts/download_testdata.sh
# または
make download-testdata
```

> **注意：** テキストデータおよびインデックスファイル（`data/*.txt`, `data/*.idx`）は `.gitignore` に登録されており、リポジトリにはコミットしないでください。

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
