package key_store

import (
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"os"
)

func IsValidFolderPath(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	if s.IsDir() == false {
		return false
	}
	return true
}

func IsPathExist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func IsValidTopiaAddr(addr tpcrtypes.Address) bool {
	_, err := addr.CryptType()
	if err != nil {
		return false
	}
	return true
}

func SetDirReadOnly(path string, dirName string) error {
	userPath, _ := os.Getwd()
	err := os.Chdir(path)
	if err != nil {
		return err
	}
	err = os.Chmod(dirName, 0544)
	if err != nil {
		return err
	}
	err = os.Chdir(userPath)
	if err != nil {
		return err
	}
	return nil
}

func SetDirPerm(path string, dirName string) error {
	userPath, _ := os.Getwd()
	err := os.Chdir(path)
	if err != nil {
		return err
	}
	err = os.Chmod(dirName, os.ModePerm)
	if err != nil {
		return err
	}
	err = os.Chdir(userPath)
	if err != nil {
		return err
	}
	return nil
}
