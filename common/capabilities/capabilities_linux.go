package capabilities

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

// AlignEffectiveCapToPermitted raises the process's effective capability set to
// match its permitted set across all OS threads. Running as root this is a no-op (effective already equals permitted).
//
// Processes calling this function and setting the effective capabilities are considered not "capability-dumb"
// according to https://man7.org/linux/man-pages/man7/capabilities.7.html
// meaning, the effective capabilities bit is not set on the process binary.
//
// Should be called on process startup/init.
func AlignEffectiveCapToPermitted() error {
	hdr := unix.CapUserHeader{Version: unix.LINUX_CAPABILITY_VERSION_3}
	var d [2]unix.CapUserData
	// The permitted set is uniform across threads (inherited at exec and never changed),
	// so reading it from the calling thread and setting effective = permitted on all
	// threads is correct.
	if err := unix.Capget(&hdr, &d[0]); err != nil {
		return err
	}
	if d[0].Effective == d[0].Permitted && d[1].Effective == d[1].Permitted {
		return nil
	}
	d[0].Effective, d[1].Effective = d[0].Permitted, d[1].Permitted

	if _, _, errno := syscall.AllThreadsSyscall(
		unix.SYS_CAPSET,
		uintptr(unsafe.Pointer(&hdr)),
		uintptr(unsafe.Pointer(&d[0])),
		0,
	); errno != 0 {
		return errno
	}
	return nil
}
