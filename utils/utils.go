package utils

import "os"

func EnsureDirExists(path string, shouldPanic bool) error {
	err := os.MkdirAll(path, 0700)
	if err != nil && !os.IsExist(err) {
		if shouldPanic {
			panic(err)
		}
		return err
	}
	return nil
}
