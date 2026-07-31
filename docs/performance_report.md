# FM-index 性能測定レポート（更新版）

## 概要

利用可能なアルゴリズムとデータ構造の拡張（`bifmindex`, `wavelet`, `waveletmatrix`, `rlbwt`）に合わせて、ベンチマークを更新しました。  
本レポートでは、まず **1MB 以下**のデータセットで機能確認を行い、その後 **妥当サイズ（8MB 級）**で性能測定を実施した結果を示します。

対象コード: [`bench_test.go`](../bench_test.go)

---

## 測定環境

| 項目 | 値 |
|---|---|
| CPU | AMD EPYC 9V74 80-Core Processor |
| OS | Linux (amd64) |
| Go | go test 実行環境 |
| 実行コマンド | `go test -run '^$' -bench 'Benchmark(Build|SearchExact|SearchRegex)_(SmallUnder1MB|ReasonableSize)' -benchmem -benchtime=1s .` |

> 注: 本実行では外部データ取得不可だったため、ベンチマークコード内のフォールバックにより合成データセット（`SyntheticJA-768KB`, `SyntheticCode-8MB`）を使用しています。

---

## 1) 1MB 以下データセットでの機能確認（SyntheticJA-768KB）

### 構築ベンチマーク

| 手法 | ns/op | MB/s | B/op | allocs/op |
|---|---:|---:|---:|---:|
| FM Doubling + Bitvectors | 876,565,241 | 0.90 | 31,399,456 | 139 |
| FM SA-IS + Bitvectors | 40,321,657 | 19.50 | 67,438,885 | 292 |
| FM SA-IS + WaveletTree | 60,839,959 | 12.93 | 92,054,209 | 2,436 |
| FM SA-IS + WaveletMatrix | 58,211,005 | 13.51 | 76,470,982 | 247 |
| FM SA-IS + RLBWT | 38,334,988 | 20.51 | 61,267,361 | 260 |
| BiFM SA-IS + Bitvectors | 79,327,013 | 9.91 | 137,206,825 | 625 |
| Stdlib SuffixArray | 16,710,959 | 47.06 | 3,145,812 | 2 |

### 検索ベンチマーク（代表パターン）

#### 完全一致 `上杉謙信`

| 手法 | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| FM Bitvectors | 175.2 | 0 | 0 |
| FM WaveletTree | 2,478 | 0 | 0 |
| FM WaveletMatrix | 2,749 | 0 | 0 |
| FM RLBWT | 152.2 | 0 | 0 |
| BiFM Bitvectors | 173.9 | 0 | 0 |
| Stdlib Lookup | 19,933 | 172,032 | 1 |

#### Star-free 正規表現 `上杉謙信|武田信玄`

| 手法 | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| FM Bitvectors | 3,236 | 2,128 | 75 |
| FM WaveletTree | 8,189 | 2,128 | 75 |
| FM WaveletMatrix | 8,572 | 2,128 | 75 |
| FM RLBWT | 3,131 | 2,128 | 75 |

---

## 2) 妥当サイズデータセットでの本測定（SyntheticCode-8MB）

### 構築ベンチマーク

| 手法 | ns/op | 秒/op | MB/s | B/op | allocs/op |
|---|---:|---:|---:|---:|---:|
| FM Doubling + Bitvectors | 37,263,303,508 | 37.26 | 0.23 | 340,353,080 | 161 |
| FM SA-IS + Bitvectors | 676,878,472 | 0.68 | 12.39 | 669,073,816 | 345 |
| FM SA-IS + WaveletTree | 856,103,310 | 0.86 | 9.80 | 971,848,480 | 3,251 |
| FM SA-IS + WaveletMatrix | 832,600,648 | 0.83 | 10.08 | 756,863,656 | 285 |
| FM SA-IS + RLBWT | 646,014,447 | 0.65 | 12.99 | 597,222,104 | 322 |
| BiFM SA-IS + Bitvectors | 1,319,737,693 | 1.32 | 6.36 | 1,321,558,256 | 687 |
| Stdlib SuffixArray | 168,079,575 | 0.17 | 49.91 | 33,554,512 | 2 |

### 完全一致検索（代表: `commit`）

| 手法 | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| FM Bitvectors | 87.53 | 0 | 0 |
| FM WaveletTree | 1,029 | 0 | 0 |
| FM WaveletMatrix | 1,056 | 0 | 0 |
| FM RLBWT | 88.56 | 0 | 0 |
| BiFM Bitvectors | 88.75 | 0 | 0 |
| Stdlib Lookup | 85,323 | 761,856 | 1 |

### Star-free 正規表現検索（代表: `commit|diff`）

| 手法 | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| FM Bitvectors | 2,131 | 1,568 | 46 |
| FM WaveletTree | 3,883 | 1,568 | 46 |
| FM WaveletMatrix | 4,096 | 1,568 | 46 |
| FM RLBWT | 2,128 | 1,568 | 46 |

---

## 3) 生成ログ + ゲノムデータでの測定

追加データ準備:

- 生成ログ: `make generate-fake-log-apache-common FAKE_LOG_SIZE=5M`（`data/fake-logs/flog_apache_common.log`, 5,293,755 bytes）
- ゲノム: `ecoli.fna` を取得して `./scripts/prepare_ecoli.sh data` 実行（`data/ecoli.txt`, 4,641,653 bytes）

実行コマンド:

`go test -run '^$' -bench 'Benchmark(Build|SearchExact|SearchRegex)_(GeneratedLogs|Genome)$' -benchmem -benchtime=1s .`

### 構築ベンチマーク

#### 生成ログ（GeneratedLog-ApacheCommon）

| 手法 | ns/op | 秒/op | MB/s | B/op | allocs/op |
|---|---:|---:|---:|---:|---:|
| FM Doubling + Bitvectors | 5,578,916,166 | 5.58 | 0.95 | 250,397,544 | 209 |
| FM SA-IS + Bitvectors | 540,469,320 | 0.54 | 9.79 | 499,425,200 | 399 |
| FM SA-IS + WaveletTree | 718,186,080 | 0.72 | 7.37 | 642,658,776 | 4,090 |
| FM SA-IS + WaveletMatrix | 688,251,320 | 0.69 | 7.69 | 519,184,328 | 256 |
| FM SA-IS + RLBWT | 567,690,974 | 0.57 | 9.33 | 659,211,136 | 1,244 |
| BiFM SA-IS + Bitvectors | 1,100,912,570 | 1.10 | 4.81 | 994,954,064 | 794 |
| Stdlib SuffixArray | 222,623,884 | 0.22 | 23.78 | 21,176,419 | 2 |

#### ゲノム（Genome-Ecoli）

| 手法 | ns/op | 秒/op | MB/s | B/op | allocs/op |
|---|---:|---:|---:|---:|---:|
| FM Doubling + Bitvectors | 8,099,701,495 | 8.10 | 0.57 | 155,539,040 | 53 |
| FM SA-IS + Bitvectors | 581,466,858 | 0.58 | 7.98 | 371,725,072 | 330 |
| FM SA-IS + WaveletTree | 743,348,660 | 0.74 | 6.24 | 588,929,072 | 1,064 |
| FM SA-IS + WaveletMatrix | 710,287,826 | 0.71 | 6.53 | 453,012,408 | 355 |
| FM SA-IS + RLBWT | 641,567,742 | 0.64 | 7.23 | 965,392,200 | 512 |
| BiFM SA-IS + Bitvectors | 1,169,226,115 | 1.17 | 3.97 | 748,291,648 | 662 |
| Stdlib SuffixArray | 310,290,880 | 0.31 | 14.96 | 18,571,368 | 2 |

### 検索ベンチマーク（代表パターン）

#### 完全一致（ログ: `HTTP/1.1` / ゲノム: `ATGAAACGC`）

| 手法 | ログ ns/op | ゲノム ns/op |
|---|---:|---:|
| FM Bitvectors | 117.0 | 130.6 |
| FM WaveletTree | 1,542 | 1,759 |
| FM WaveletMatrix | 1,582 | 2,087 |
| FM RLBWT | 352.6 | 595.0 |
| BiFM Bitvectors | 116.8 | 130.2 |
| Stdlib Lookup | 16,083 | 298.3 |

#### Star-free 正規表現（ログ: `GET\|POST` / ゲノム: `ATGAAACGC\|GTTACCTGCC`）

| 手法 | ログ ns/op | ゲノム ns/op |
|---|---:|---:|
| FM Bitvectors | 1,730 | 3,297 |
| FM WaveletTree | 2,860 | 7,145 |
| FM WaveletMatrix | 2,894 | 7,594 |
| FM RLBWT | 1,877 | 4,264 |

---

## 4) まとめ

- **構築**: Doubling は 8MB で極端に遅く、実用上は SA-IS 系が有利。  
- **Occ 構造**: このデータでは `Bitvectors` と `RLBWT` が検索で同等に高速、`WaveletTree/Matrix` は検索が遅め。  
- **BiFM-index**: 完全一致検索性能は FM(Bitvectors) と同等だが、構築コストはおおむね 2 倍。  
- **Stdlib**: 構築は最速だが、検索時は FM 系より大幅に遅く、`Lookup` のアロケーションコストも大きい。  
- **追加測定**: 生成ログ（5MB 級）と E. coli ゲノム（4.6MB 級）でも同傾向で、検索は FM Bitvectors / BiFM が安定して高速、正規表現は RLBWT が次点。
