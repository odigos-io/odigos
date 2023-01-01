package main

import "fmt"

func main() {
	fmt.Println("I like fruit loops")

	j := 0
	for i := 0; i < 10; i++ {
		fmt.Println(i, j-i)
	}
}
