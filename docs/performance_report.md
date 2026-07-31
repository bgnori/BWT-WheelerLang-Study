# FM-index 性能測定レポート（再測定版）

## 概要

メモリ消費とインデックスファイルサイズの測定を追加したため、ベンチマークを再実行しました。  
本レポートでは、従来の `go test -bench` による構築/検索比較に加えて、CLI 実行時の **ピークRSS（近似）** と **生成インデックスサイズ** を追記しています。

対象コード: [`bench_test.go`](../bench_test.go), [`Makefile`](../Makefile)

---

## 測定環境

| 項目 | 値 |
|---|---|
| CPU | AMD EPYC 7763 64-Core Processor |
| OS | Linux (amd64) |
| Go | go test 実行環境 |
| ベンチコマンド1 | `go test -run '^$' -bench 'Benchmark(Build|SearchExact|SearchRegex)_(SmallUnder1MB|ReasonableSize)' -benchmem -benchtime=1s .` |
| ベンチコマンド2 | `go test -run '^$' -bench 'Benchmark(Build|SearchExact|SearchRegex)_(GeneratedLogs|Genome)$' -benchmem -benchtime=1s .` |

注:

- 生成ログは外部生成ツール未導入のため、フォールバックの `SyntheticLog-5MB` を使用。
- ゲノムは `./scripts/prepare_ecoli.sh data` 実行後の `Genome-Ecoli` を使用。

---

## 1) 1MB 以下データセット（Kenshin<=1MB）

### 構築ベンチマーク

| 手法 | ns/op | MB/s | B/op | allocs/op |
|---|---:|---:|---:|---:|
| FM Doubling + Bitvectors | 286,553,701 | 1.55 | 26,988,168 | 358 |
| FM SA-IS + Bitvectors | 39,972,009 | 11.11 | 52,666,621 | 507 |
| FM SA-IS + WaveletTree | 55,665,690 | 7.98 | 57,360,296 | 3,288 |
| FM SA-IS + WaveletMatrix | 56,542,398 | 7.85 | 48,504,888 | 216 |
| FM SA-IS + RLBWT | 44,568,809 | 9.96 | 59,403,144 | 1,230 |
| BiFM SA-IS + Bitvectors | 87,003,523 | 5.10 | 105,742,857 | 1,016 |
| Stdlib SuffixArray | 22,093,137 | 20.10 | 2,080,851 | 3 |

### 検索ベンチマーク（代表パターン）

#### 完全一致 `上杉謙信`

| 手法 | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| FM Bitvectors | 187.6 | 0 | 0 |
| FM WaveletTree | 2,152 | 0 | 0 |
| FM WaveletMatrix | 2,097 | 0 | 0 |
| FM RLBWT | 432.2 | 0 | 0 |
| BiFM Bitvectors | 198.9 | 0 | 0 |
| Stdlib Lookup | 307.5 | 80 | 1 |

#### Star-free 正規表現 `上杉謙信|武田信玄`

| 手法 | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| FM Bitvectors | 4,618 | 2,128 | 75 |
| FM WaveletTree | 8,350 | 2,128 | 75 |
| FM WaveletMatrix | 9,459 | 2,128 | 75 |
| FM RLBWT | 4,737 | 2,128 | 75 |

---

## 2) 妥当サイズデータセット（GitSource）

### 構築ベンチマーク

| 手法 | ns/op | 秒/op | MB/s | B/op | allocs/op |
|---|---:|---:|---:|---:|---:|
| FM Doubling + Bitvectors | 30,963,843,996 | 30.96 | 0.32 | 631,903,200 | 419 |
| FM SA-IS + Bitvectors | 1,765,752,076 | 1.77 | 5.58 | 1,121,450,128 | 720 |
| FM SA-IS + WaveletTree | 1,697,028,140 | 1.70 | 5.80 | 1,234,510,600 | 5,802 |
| FM SA-IS + WaveletMatrix | 1,601,747,640 | 1.60 | 6.15 | 992,097,064 | 378 |
| FM SA-IS + RLBWT | 1,543,003,363 | 1.54 | 6.38 | 1,267,681,304 | 2,309 |
| BiFM SA-IS + Bitvectors | 3,313,446,607 | 3.31 | 2.97 | 2,250,846,512 | 1,442 |
| Stdlib SuffixArray | 555,611,138 | 0.56 | 17.73 | 39,403,608 | 3 |

### 完全一致検索（代表: `commit`）

| 手法 | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| FM Bitvectors | 101.9 | 0 | 0 |
| FM WaveletTree | 951.0 | 0 | 0 |
| FM WaveletMatrix | 1,144 | 0 | 0 |
| FM RLBWT | 308.0 | 0 | 0 |
| BiFM Bitvectors | 85.92 | 0 | 0 |
| Stdlib Lookup | 15,603 | 81,920 | 1 |

### Star-free 正規表現検索（代表: `commit|diff`）

| 手法 | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| FM Bitvectors | 3,543 | 1,568 | 46 |
| FM WaveletTree | 6,201 | 1,568 | 46 |
| FM WaveletMatrix | 5,106 | 1,568 | 46 |
| FM RLBWT | 3,888 | 1,568 | 46 |

---

## 3) 追加データセット（SyntheticLog-5MB / Genome-Ecoli）

実行コマンド:

`go test -run '^$' -bench 'Benchmark(Build|SearchExact|SearchRegex)_(GeneratedLogs|Genome)$' -benchmem -benchtime=1s .`

### 構築ベンチマーク

#### ログ（SyntheticLog-5MB）

| 手法 | ns/op | 秒/op | MB/s | B/op | allocs/op |
|---|---:|---:|---:|---:|---:|
| FM Doubling + Bitvectors | 19,804,180,237 | 19.80 | 0.26 | 218,259,592 | 173 |
| FM SA-IS + Bitvectors | 400,567,601 | 0.40 | 13.09 | 398,311,536 | 342 |
| FM SA-IS + WaveletTree | 612,032,664 | 0.61 | 8.57 | 575,601,456 | 3,364 |
| FM SA-IS + WaveletMatrix | 525,329,654 | 0.53 | 9.98 | 447,762,824 | 270 |
| FM SA-IS + RLBWT | 385,839,496 | 0.39 | 13.59 | 347,889,720 | 307 |
| BiFM SA-IS + Bitvectors | 840,728,340 | 0.84 | 6.24 | 808,403,184 | 687 |
| Stdlib SuffixArray | 100,771,032 | 0.10 | 52.03 | 20,971,611 | 2 |

#### ゲノム（Genome-Ecoli）

| 手法 | ns/op | 秒/op | MB/s | B/op | allocs/op |
|---|---:|---:|---:|---:|---:|
| FM Doubling + Bitvectors | 8,814,013,176 | 8.81 | 0.53 | 155,538,792 | 54 |
| FM SA-IS + Bitvectors | 636,361,430 | 0.64 | 7.29 | 371,724,816 | 330 |
| FM SA-IS + WaveletTree | 813,924,285 | 0.81 | 5.70 | 588,928,936 | 1,063 |
| FM SA-IS + WaveletMatrix | 735,606,678 | 0.74 | 6.31 | 453,012,264 | 354 |
| FM SA-IS + RLBWT | 751,131,472 | 0.75 | 6.18 | 965,392,104 | 511 |
| BiFM SA-IS + Bitvectors | 1,252,923,656 | 1.25 | 3.70 | 748,291,120 | 662 |
| Stdlib SuffixArray | 272,837,813 | 0.27 | 17.01 | 18,571,348 | 2 |

### 検索ベンチマーク（代表パターン）

#### 完全一致（ログ: `HTTP/1.1` / ゲノム: `ATGAAACGC`）

| 手法 | ログ ns/op | ゲノム ns/op |
|---|---:|---:|
| FM Bitvectors | 121.4 | 130.0 |
| FM WaveletTree | 1,442 | 1,751 |
| FM WaveletMatrix | 1,277 | 1,742 |
| FM RLBWT | 126.2 | 592.2 |
| BiFM Bitvectors | 124.2 | 137.1 |
| Stdlib Lookup | 81,142 | 475.8 |

#### Star-free 正規表現（ログ: `GET\|POST` / ゲノム: `ATGAAACGC\|GTTACCTGCC`）

| 手法 | ログ ns/op | ゲノム ns/op |
|---|---:|---:|
| FM Bitvectors | 1,978 | 4,576 |
| FM WaveletTree | 2,326 | 8,028 |
| FM WaveletMatrix | 2,998 | 8,309 |
| FM RLBWT | 2,014 | 5,993 |

---

## 4) メモリ消費とインデックスサイズ（CLI 実測）

`bwtsearch` CLI で FM/SAIS と stdlib SuffixArray を比較。  
ピークRSSは `ps` サンプリングによる近似値（kB）です。

### 構築時メトリクス

| データセット | 手法 | elapsed_sec | peak_rss_kb | index_bytes |
|---|---|---:|---:|---:|
| Kenshin | FM/SAIS | 0.158 | 35,536 | 8,831,512 |
| Kenshin | Stdlib/SuffixArray | 0.124 | 10,636 | 2,204,893 |
| GitSource | FM/SAIS | 7.889 | 574,916 | 216,694,136 |
| GitSource | Stdlib/SuffixArray | 5.033 | 78,628 | 57,006,331 |
| Ecoli | FM/SAIS | 2.273 | 273,684 | 31,333,594 |
| Ecoli | Stdlib/SuffixArray | 1.340 | 37,712 | 25,746,339 |
| OsativaChr1 | FM/SAIS | 31.381 | 2,329,996 | 297,490,092 |
| OsativaChr1 | Stdlib/SuffixArray | 24.428 | 266,960 | 257,616,375 |

---

## 5) まとめ

- **構築速度**: すべてのデータセットで Doubling は遅く、実用上は SA-IS 系が優位。
- **検索速度（完全一致）**: `Bitvectors` と `BiFM(Bitvectors)` が最速帯。`WaveletTree/Matrix` は 1 桁以上遅い傾向。
- **検索速度（正規表現）**: `Bitvectors` と `RLBWT` が近い性能で、`WaveletTree/Matrix` は遅め。
- **FM vs Stdlib（go test）**: 構築では stdlib が有利だが、検索（特にログ系）では FM 系が大幅に高速。
- **FM vs Stdlib（CLI 実測）**: FM/SAIS はインデックスサイズとピークRSSが大きくなる一方、データセットによっては検索時に有利（例: OsativaChr1）。
