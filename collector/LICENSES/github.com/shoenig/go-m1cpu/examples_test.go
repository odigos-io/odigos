package m1cpu

import (
	"fmt"
)

func ExampleIsAppleSilicon() {
	value := IsAppleSilicon()
	fmt.Println(value)
}

func ExampleModelName() {
	value := ModelName()
	fmt.Println(value)
}

func ExamplePCoreHz() {
	value := PCoreHz()
	fmt.Println(value)
}

func ExampleECoreHz() {
	value := ECoreHz()
	fmt.Println(value)
}

func ExamplePCoreGHz() {
	value := PCoreGHz()
	fmt.Println(value)
}

func ExampleECoreGHz() {
	value := ECoreGHz()
	fmt.Println(value)
}

func ExamplePCoreCount() {
	value := PCoreCount()
	fmt.Println(value)
}

func ExampleECoreCount() {
	value := ECoreCount()
	fmt.Println(value)
}

func ExamplePCoreCache() {
	l1inst, l1data, l2 := PCoreCache()
	fmt.Println(l1inst, l1data, l2)
}

func ExampleECoreCache() {
	l1inst, l1data, l2 := ECoreCache()
	fmt.Println(l1inst, l1data, l2)
}
