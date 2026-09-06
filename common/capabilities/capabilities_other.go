//go:build !linux

package capabilities

// AlignEffectiveCapToPermitted is a no-op on non-Linux platforms.
func AlignEffectiveCapToPermitted() error {
	return nil
}
