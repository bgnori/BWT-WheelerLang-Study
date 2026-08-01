# 性能測定計画（改定Aフェーズ）

更新日: 2026-08-01

## 目的

全組合せの総当たりを避けつつ、次の判断材料を短期間で得る。

- インデックス構築時間
- 構築時の最大メモリ（peak RSS 近似）
- 検索応答速度（p50/p95）
- 検索時メモリ（peak RSS 近似）
- インデックスファイルサイズ
- 未計測 Occ 構造（`rrr`, `eliasfano`, `poppy`, `dynamic`）の当たり

本計画は、従来の Phase A/B を統合し、Phase A のデータセットに対して B 相当のクエリ設計を適用する。
Phase C（external storage 詳細比較）は本ドキュメントの対象外とする。

## 改定方針

1. `Osativa chr1` は改定Aフェーズから外す
2. 英文評価の代表として `Moby Dick` を採用する
3. ゲノム代表は `Ecoli` のみとし、重複を避ける
4. クエリ設計は B 相当（高頻度・低頻度・0ヒット）に統一する
5. Occ は既存 3 種に加えて未計測 4 種を入れ、まずは当たりをつける

## 対象データセット（改定A）

- `Kenshin`（日本語、小〜中規模）
- `MobyDick`（英語、小〜中規模）
- `GitSource`（ソースコード、中規模）
- `Ecoli`（ゲノム、中規模）

補足:

- `MobyDick` は英語文に対する機能性と比較可能性を担保するため追加
- `Osativa chr1` は規模が大きく、今回の「広く浅いスクリーニング」には重いため除外

## 対象インデックス構成

アルゴリズムは比較軸を絞るため `--algo sais` 固定とする（suffixarray を除く）。

### 比較本体（必須）

- `FM SA-IS + rlbwt`
- `FM SA-IS + bitvectors`
- `FM SA-IS + waveletmatrix`
- `Stdlib suffixarray`

### 当たり付け（追加）

- `FM SA-IS + rrr`
- `FM SA-IS + eliasfano`
- `FM SA-IS + poppy`
- `FM SA-IS + dynamic`

注:

- `wavelet` は今回の改定Aでは省略（`waveletmatrix` と役割が近く、組合せ削減を優先）
- `bifmindex` は今回の改定Aでは省略（まず単方向FMの比較を優先）

## クエリ設計（B相当を統合）

各データセットで、検索クエリを以下 4 種に固定する。

1. 完全一致（高頻度語）
2. 完全一致（低頻度語）
3. 完全一致（0ヒット語）
4. OR 正規表現（`a|b` 形式、FMのみ実施）

### 推奨クエリ例

- Kenshin
  - 高頻度: `謙信`
  - 低頻度: `越前`
  - 0ヒット: `織田信長公記`（データ内不在を事前確認）
  - OR: `上杉謙信|武田信玄`
- MobyDick
  - 高頻度: `whale`
  - 低頻度: `harpooner`
  - 0ヒット: `microservice`
  - OR: `white|whale`
- GitSource
  - 高頻度: `struct`
  - 低頻度: `chdir-notify`
  - 0ヒット: `kubernetes`
  - OR: `commit|diff`
- Ecoli
  - 高頻度: `AAAAAA`
  - 低頻度: `ATGAAACGC`
  - 0ヒット: `NNNNNNNN`
  - OR: `ATGAAACGC|GTTACCTGCC`

注:

- 0ヒット語は事前に 1 回だけ確認する
- suffixarray では OR 正規表現を実施せず、完全一致 3 種のみ実施する

## 測定項目

### 構築フェーズ

- `build_elapsed_sec`
- `build_peak_rss_kb`（近似）
- `index_bytes`

### 検索フェーズ

- `search_mode`（cold / warm）
- `search_elapsed_ms`
- `search_p50_ms`
- `search_p95_ms`
- `search_peak_rss_kb`（近似）

### 追加記録（推奨）

- `hits`（ヒット件数）
- `backend`（fm / suffixarray）
- `occ`
- `dataset_bytes`

## 実行回数と集計

- 構築: 各条件 3 回（中央値採用）
- 検索: 各クエリ条件 10 回（中央値 + p95）
- 冷温分離:
  - cold: プロセス起動直後 1 回目
  - warm: 同一条件で連続 9 回

## ケース数（概算）

- 構築ケース: 4 datasets x 8 configs = 32
- 検索ケース（完全一致）: 4 datasets x 8 configs x 3 queries = 96
- 検索ケース（OR, FMのみ）: 4 datasets x 7 fm-configs x 1 query = 28

合計 156 条件。
これを 2 段階で実行する。

1. スクリーニング: 新Occ 4種はまず `MobyDick` と `GitSource` のみ
2. 拡張: 有望な新Occのみ `Kenshin` と `Ecoli` へ展開

この分割により、初手の計測量を抑えつつ新Occの当たりを得る。

## 実行順序（推奨）

1. 比較本体 4 構成を 4 データセットで一巡
2. 新Occ 4 構成を `MobyDick` と `GitSource` に限定して追加
3. 新Occの上位 1〜2 構成のみ `Kenshin` と `Ecoli` に展開
4. 集計して「速度優先」「メモリ優先」「サイズ優先」の候補を各 1 つ選定

## 実行ハンドル

- 各実験は同一のラッパーから起動し、構築・検索・CSV 収集を一元管理する。
- 目標として `make bench-phase-a` もしくは `scripts/run_phase_a_bench.sh` のような単一エントリポイントを用意し、実行内容を再現可能にする。
- 1 条件ごとに `run_id` を付与し、再実行時にも差分を追えるようにしておく。

## 事前確認

- 0ヒット語は実測前に 1 回だけ確認し、`hits = 0` であることを確認する。見つかった場合は別のクエリに差し替える。
- OR 正規表現は FM 側のみで実施し、suffixarray では実施しない。実行前に対象パターンが CLI 上で受理できることを確認する。
- 各データセットのファイルサイズ・文字数を記録し、`dataset_bytes` の基準を固定する。

## RSS・メモリ測定の標準化

- peak RSS は `/proc/<pid>/status` の `VmRSS` をポーリングし、測定開始から終了までの最大値を採用する。
- サンプリング間隔は 100ms〜500ms とし、構築・検索の両フェーズで同じ基準を用いる。
- 親プロセスのみでなく、必要に応じて子プロセスも追跡対象に含める。追跡対象を明記しておく。
- 0ヒット語や Regex 実行では一時的にメモリピークが上がることがあるため、同じログ収集ルールを適用する。

## 候補選定ルール

- 最終候補は「単一指標最良」ではなく、速度・メモリ・サイズの Pareto front から選ぶ。
- 速度優先では、検索 `p50/p95` が最小かつ構築時間が最小の構成を優先する。
- メモリ優先では、構築/検索の `peak RSS` と `index_bytes` が小さい構成を優先する。
- サイズ優先では `index_bytes` が最小の構成を優先し、速度の劣化が大きすぎないものを採用する。
- 新Occ は、各データセットで少なくとも 1 つの指標で条件付き有望と判断できる場合に次段階へ進める。

## 収集フォーマット（CSV）

列は以下で統一する。

- `dataset`
- `dataset_bytes`
- `algo`
- `occ`
- `backend`
- `query_type` (`exact_high|exact_low|exact_miss|regex_or`)
- `query`
- `search_mode` (`cold|warm`)
- `build_elapsed_sec`
- `build_peak_rss_kb`
- `index_bytes`
- `search_elapsed_ms`
- `search_p50_ms`
- `search_p95_ms`
- `search_peak_rss_kb`
- `hits`
- `run_id`

## 判定ルール（改定Aの出口条件）

改定Aフェーズの完了条件を以下とする。

1. 各データセットで「速度最良の構成」を 1 つ特定できる
2. 各データセットで「メモリ最良の構成」を 1 つ特定できる
3. 新Occ（`rrr`, `eliasfano`, `poppy`, `dynamic`）について、
   - 明確に不利（速度/メモリ/サイズのいずれでも優位がない）
   - 条件付きで有望
   のどちらかを判定できる
4. 次段階（大規模・external比較）に持ち込む候補を最大 2 構成に絞れる

## 注意事項

- 本環境では `/usr/bin/time` が使えないため、peak RSS は `ps` もしくは `/proc/<pid>/status` のサンプリングで近似値を取得する
- 絶対値より相対比較を重視する
- 既存の結果レポートは [performance_report.md](performance_report.md) に記録し、本書は計画専用として運用する
