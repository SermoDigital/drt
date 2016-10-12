// +build !windows,!plan9,!solaris

package drt

import (
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

func mmap(file *os.File) ([]byte, error) {
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	b, err := unix.Mmap(int(file.Fd()), 0, int(stat.Size()), unix.PROT_READ, unix.MAP_SHARED)

	// TODO: MADV_DONTNEED?
	err = madvise(b, unix.MADV_RANDOM)
	if err != nil {
		return nil, err
	}
	return b, err
}

func munmap(b []byte) error {
	return unix.Munmap(b)
}

func madvise(b []byte, advice int) (err error) {
	_, _, e1 := unix.Syscall(unix.SYS_MADVISE, uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)), uintptr(advice))
	if e1 != 0 {
		err = e1
	}
	return
}
