package common

// +kubebuilder:validation:Enum=glibc;musl
type LibCType string

const (
	Glibc LibCType = "glibc"
	Musl  LibCType = "musl"
)
