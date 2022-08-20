//go:build !windows

package file_key_store

import (
	"os"
	"path/filepath"
	"syscall"
)

func (f *FileKeyStore) lockSH() error {
	file, err := os.OpenFile(filepath.Join(f.fileFolderPath, lockFileName), os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	return syscall.Flock(int(file.Fd()), syscall.LOCK_SH)

}

func (f *FileKeyStore) lockEX() error {
	file, err := os.OpenFile(filepath.Join(f.fileFolderPath, lockFileName), os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	return syscall.Flock(int(file.Fd()), syscall.LOCK_EX)
}

func (f *FileKeyStore) unlock() error {
	file, err := os.Open(filepath.Join(f.fileFolderPath, lockFileName))
	if err != nil {
		return err
	}
	defer file.Close()

	return syscall.Flock(int(file.Fd()), syscall.LOCK_UN)

}
