//go:build windows

package file_key_store

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

const (
	LOCKFILE_EXCLUSIVE_LOCK = 0x00000002
)

var (
	modkernel32      = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = modkernel32.NewProc("LockFileEx")
	procUnlockFileEx = modkernel32.NewProc("UnlockFileEx")
)

func (f *FileKeyStore) lockSH() error {
	file, err := os.OpenFile(filepath.Join(f.fileFolderPath, lockFileName), os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	r1, errNo := wlock(file, 0x0)
	return isWError(r1, errNo)
}

func (f *FileKeyStore) lockEX() error {
	file, err := os.OpenFile(filepath.Join(f.fileFolderPath, lockFileName), os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	r1, errNo := wlock(file, LOCKFILE_EXCLUSIVE_LOCK)
	return isWError(r1, errNo)
}

func (f *FileKeyStore) unlock() error {
	file, err := os.Open(filepath.Join(f.fileFolderPath, lockFileName))
	if err != nil {
		return err
	}
	defer file.Close()

	r1, _, errNo := syscall.SyscallN(
		procUnlockFileEx.Addr(),
		file.Fd(),
		uintptr(0),
		uintptr(1),
		uintptr(0),
		uintptr(unsafe.Pointer(&syscall.Overlapped{})),
	)
	return isWError(r1, errNo)
}

func wlock(fp *os.File, flags uintptr) (uintptr, syscall.Errno) {
	r1, _, errNo := syscall.SyscallN(
		procLockFileEx.Addr(),
		fp.Fd(),
		flags,
		uintptr(0),
		uintptr(1),
		uintptr(0),
		uintptr(unsafe.Pointer(&syscall.Overlapped{})),
	)
	return r1, errNo
}

func isWError(r1 uintptr, errNo syscall.Errno) error {
	if r1 != 1 {
		if errNo != 0 {
			return errors.New(errNo.Error())
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}
