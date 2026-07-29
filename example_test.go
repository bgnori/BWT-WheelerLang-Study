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
