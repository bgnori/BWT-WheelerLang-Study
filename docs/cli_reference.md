# CLI Reference

## 共通

```bash
textindex <command> [args]
```

サポートコマンド:

- `build`
- `build-multi`
- `info`
- `graph`
- `browse`
- `search`
- `web`

---

## `build` — 単一ファイルからインデックス構築

```bash
textindex build [--algo doubling|sais|suffixarray|bifmindex] \
  [--occ bitvectors|wavelet|waveletmatrix|rlbwt] <input-file> <index-file>
```

- `--algo`（既定: `sais`）
  - `doubling`: FM-index（前置倍加法）
  - `sais`: FM-index（SA-IS）
  - `suffixarray`: Go 標準ライブラリ Suffix Array（リテラル検索専用）
  - `bifmindex`: 双方向 FM-index（リテラル検索）
- `--occ`（既定: `rlbwt`）
  - `bitvectors`
  - `wavelet`
  - `waveletmatrix`
  - `rlbwt`

例:

```bash
textindex build data/moby_dick.txt data/moby_dick.idx
textindex build --algo sais --occ waveletmatrix data/moby_dick.txt data/moby_dick.idx
textindex build --algo suffixarray data/moby_dick.txt data/moby_dick.saidx
textindex build --algo bifmindex --occ rlbwt data/moby_dick.txt data/moby_dick.bidx
```

---

## `build-multi` — 複数ファイルからインデックス構築

```bash
textindex build-multi [--algo doubling|sais|suffixarray|bifmindex] \
  [--occ bitvectors|wavelet|waveletmatrix|rlbwt] <index-file> <file1> [file2 ...]
```

入力ファイルは内部で連結されます（既定セパレータは改行）。

例:

```bash
find data/git-src -type f \( -name "*.c" -o -name "*.h" \) | sort | \
  xargs textindex build-multi --algo sais data/git.idx
```

---

## `info` — インデックス情報表示

```bash
textindex info <index-file>
```

主な表示内容:

- backend 種別
- text length
- FM-index の場合: SA length / alphabet size / occ structure / bwt runs

---

## `graph` — Wheeler グラフ出力（Mermaid）

```bash
textindex graph [--max-nodes N] [--markdown=true|false] <index-file>
```

- `--max-nodes`: 描画ノード数上限（`0` で全件）
- `--markdown`: Mermaid フェンス付き出力（既定 `true`）

> `graph` は FM-index 専用です（`suffixarray` と `bifmindex` では利用不可）。

---

## `browse` — 対話ブラウズ

```bash
textindex browse <index-file> [--show N] [--context N]
```

- 端末 TTY ではインタラクティブ UI
- 非 TTY では行入力モード

---

## `search` — 検索

```bash
textindex search [--limit N] [--context N] [--positions] <index-file> <pattern>
```

- `--limit`（既定: `20`）: 最大件数（`0` 以下で無制限）
- `--context`（既定: `80`）: 前後表示文字数
- `--positions`: 位置のみ出力

バックエンドごとのパターン解釈:

- FM-index (`doubling` / `sais`): 星なし正規表現
- Suffix Array (`suffixarray`): リテラル検索
- 双方向 FM-index (`bifmindex`): リテラル検索

FM-index で拒否される代表例:

- `*`
- `+`
- `{n,}`

---

## `web` — 検索 Web UI

```bash
textindex web [--index FILE] [--addr ADDR] [--limit N] [--context N] [--min-chars N]
```

- `--index`（既定: `data/moby_dick.idx`）
- `--addr`（既定: `:8080`）
- `--limit`（既定: `20`）
- `--context`（既定: `80`）
- `--min-chars`（既定: `4`、1未満は1に補正）

起動後、`http://localhost:8080` を開きます。
