package main

import (
	"fmt"
	"strings"
)

func main() {
	s := "prefix this should be all you see"
	trimmed := strings.TrimPrefix(s, "prefix")
	if trimmed != "this should be all you see" {
		fmt.Println("oh noes")
	}

	fmt.Println(trimmed)
}
