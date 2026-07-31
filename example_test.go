package bwtsearch

import (
	"fmt"
	"os"
)

func ExampleBuild() {
	idx := Build([]byte("abracadabra"))

	fmt.Println(idx.Count([]byte("abra")))
	fmt.Println(idx.Count([]byte("xyz")))
	// Output:
	// 2
	// 0
}

func ExampleSearch() {
	idx := Build([]byte("abracadabra"))

	res, err := Search(idx, "abra|cad", 0)
	if err != nil {
		fmt.Println("error")
		return
	}

	fmt.Println(res.TotalCount)
	fmt.Println(len(res.Positions(idx)))
	// Output:
	// 3
	// 3
}

func ExampleBuildWithAlgorithm() {
	idx := BuildWithAlgorithm([]byte("banana"), AlgorithmSAIS)
	fmt.Println(idx.Count([]byte("ana")))
	// Output:
	// 2
}

func ExampleIndex_Save() {
	idx := Build([]byte("abracadabra"))

	file, err := os.CreateTemp("", "bwtsearch-example-*.idx")
	if err != nil {
		fmt.Println("error")
		return
	}
	path := file.Name()
	file.Close()
	defer os.Remove(path)

	if err := idx.Save(path); err != nil {
		fmt.Println("error")
		return
	}

	loaded, err := Load(path)
	if err != nil {
		fmt.Println("error")
		return
	}

	fmt.Println(loaded.Count([]byte("abra")))
	// Output:
	// 2
}

func ExampleLoad() {
	idx := Build([]byte("banana"))

	file, err := os.CreateTemp("", "bwtsearch-example-*.idx")
	if err != nil {
		fmt.Println("error")
		return
	}
	path := file.Name()
	file.Close()
	defer os.Remove(path)

	if err := idx.Save(path); err != nil {
		fmt.Println("error")
		return
	}

	loaded, err := Load(path)
	if err != nil {
		fmt.Println("error")
		return
	}

	fmt.Println(loaded.TextLen())
	// Output:
	// 6
}

// ExampleBuild_japanese demonstrates building and searching an FM-index over
// UTF-8 Japanese text.  "上杉謙信" (Uesugi Kenshin) is a famous warlord of
// the Sengoku period; the index finds both occurrences in the sentence.
func ExampleBuild_japanese() {
	// 上杉謙信 appears twice: once at byte 15, once at byte 63.
	text := "武田信玄と上杉謙信は戦国時代の名将である。上杉謙信は越後の虎と呼ばれた。"
	idx := Build([]byte(text))

	fmt.Println(idx.Count([]byte("上杉謙信")))
	fmt.Println(idx.Count([]byte("徳川家康")))
	// Output:
	// 2
	// 0
}

// ExampleBuildWithOptions demonstrates building an FM-index with an explicit
// suffix-array algorithm and a Wavelet Tree occurrence array.
func ExampleBuildWithOptions() {
	idx := BuildWithOptions([]byte("abracadabra"), AlgorithmSAIS, OccWaveletTree)
	fmt.Println(idx.Count([]byte("abra")))
	fmt.Println(idx.Count([]byte("xyz")))
	// Output:
	// 2
	// 0
}

func ExampleIndex_Append() {
	idx := Build([]byte("hello"))
	_ = idx.Append([]byte(" world"))
	fmt.Println(idx.Count([]byte("hello world")))
	// Output:
	// 1
}
