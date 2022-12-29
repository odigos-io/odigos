package allocator

import (
	"github.com/keyval-dev/odigos/odiglet/pkg/allocator/debug/elf"
)

// Fix section table offset
// Go binaries place the section table header at the begging of the file (after the segments table)
// Allocator moves the section table to the end of the file and therefore the file header need to be patched
// The shoff field in the header points to the old location (start of the file) we change it to point to the new location
// (end of file) 40 - 48 is the relevant bytes in the header (fixed for 64bit, other values for 32bit)
func applyFixForGoBinaries(f *elf.File, shoff uint64, output []byte) []byte {
	b := make([]byte, 8)
	f.ByteOrder.PutUint64(b, shoff)
	j := 0
	for i := 40; i < 48; i++ {
		output[i] = b[j]
		j++
	}

	return output
}
