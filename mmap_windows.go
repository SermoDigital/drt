package drt

import (
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

// snagged from https://github.com/boltdb/bolt/blob/master/bolt_windows.go

func mmap(file *os.File) ([]byte, error) {
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	sz := int(stat.Size())

	// Open a file mapping handle.
	sizelo := uint32(sz >> 32)
	sizehi := uint32(sz) & 0xffffffff
	h, errno := syscall.CreateFileMapping(syscall.Handle(file.Fd()), nil, syscall.PAGE_READONLY, sizelo, sizehi, nil)
	if h == 0 {
		return nil, os.NewSyscallError("CreateFileMapping", errno)
	}

	// Create the memory map.
	addr, errno := syscall.MapViewOfFile(h, syscall.FILE_MAP_READ, 0, 0, uintptr(sz))
	if addr == 0 {
		return nil, os.NewSyscallError("MapViewOfFile", errno)
	}

	// Close mapping handle.
	if err := syscall.CloseHandle(syscall.Handle(h)); err != nil {
		return nil, os.NewSyscallError("CloseHandle", err)
	}
	switch runtime.GOARCH {
	default:
		return ((*[0xFFFFFFFFFFFF]byte)(unsafe.Pointer(addr)))[:], nil
	case "ppc", "386", "arm":
		return ((*[0x7FFFFFFF]byte)(unsafe.Pointer(addr)))[:], nil
	}
}

func munmap(b []byte) error {
	addr := (uintptr)(unsafe.Pointer(&b[0]))
	if err := syscall.UnmapViewOfFile(addr); err != nil {
		return os.NewSyscallError("UnmapViewOfFile", err)
	}
	return nil
}
