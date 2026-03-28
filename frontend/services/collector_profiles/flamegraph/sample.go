package flamegraph

// Sample is one profile stack aggregate: frame names root-first and total value (e.g. sample count).
type Sample struct {
	Stack []string
	Value int64
}
