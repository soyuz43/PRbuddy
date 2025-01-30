// ./coreutils/fileutils.go 
package coreutils

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func WriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Atomic write with file locking
	file, err := os.CreateTemp(dir, "tmp-")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())

	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("file lock failed: %w", err)
	}
	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}

	if err := os.Rename(file.Name(), path); err != nil {
		return fmt.Errorf("atomic rename failed: %w", err)
	}

	return nil
}

func ReadFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_SH); err != nil {
		return nil, fmt.Errorf("file lock failed: %w", err)
	}
	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)

	return os.ReadFile(path)
}