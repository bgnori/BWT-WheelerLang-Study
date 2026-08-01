# Wheeler Graph Mermaid Examples

このファイルは、`textindex graph` が出力する Wheeler graph の Mermaid 例です。

## 例1: `banana` 全ノード表示

入力テキスト:

```text
banana
```

生成コマンド:

```bash
printf 'banana' > /tmp/wheeler_sample.txt
./textindex build /tmp/wheeler_sample.txt /tmp/wheeler_sample.idx
./textindex graph --max-nodes 0 /tmp/wheeler_sample.idx
```

出力例:

```mermaid
flowchart LR
  %% Node order is the Wheeler order (SA rank).
  n0["0: SA=6 $"]
  n1["1: SA=5 a$"]
  n2["2: SA=3 ana$"]
  n3["3: SA=1 anana$"]
  n4["4: SA=0 banana$"]
  n5["5: SA=4 na$"]
  n6["6: SA=2 nana$"]
  n0 --"a"--> n1
  n1 --"n"--> n5
  n2 --"n"--> n6
  n3 --"b"--> n4
  n4 --"$"--> n0
  n5 --"a"--> n2
  n6 --"a"--> n3
```

## 例2: ノード制限あり（`--max-nodes 4`）

生成コマンド:

```bash
./textindex graph --max-nodes 4 /tmp/wheeler_sample.idx
```

出力例:

```mermaid
flowchart LR
  %% Node order is the Wheeler order (SA rank).
  n0["0: SA=6 $"]
  n1["1: SA=5 a$"]
  n2["2: SA=3 ana$"]
  n3["3: SA=1 anana$"]
  n0 --"a"--> n1
  hidden[("... 3 more nodes omitted")]
  note["3 edges to omitted nodes"]
  note -.-> hidden
```
