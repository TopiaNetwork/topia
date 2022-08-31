//go:build !windows

package file_key_store

import (
	"os"
	"syscall"
)

func isPIDAlive(pID int) (bool, error) {
	process, err := os.FindProcess(pID)
	if err != nil {
		return false, err
	}

	err = process.Signal(syscall.Signal(0))
	if err != nil {
		return false, err
	}
	return true, nil
}
