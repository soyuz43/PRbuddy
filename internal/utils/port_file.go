// internal/utils/port_file.go
package utils

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

const (
	appName      = "prbuddy-go"
	portFileName = "port"
	filePerm     = 0600 // rw-------
	dirPerm      = 0700 // rwx------
)

// EnsureAppCacheDir creates and validates the application cache directory
func EnsureAppCacheDir() error {
	cacheDir, err := getAppCacheDirPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(cacheDir, dirPerm); err != nil {
		return fmt.Errorf("failed to create application directory: %w", err)
	}

	return verifyDirectoryPermissions(cacheDir)
}

func getAppCacheDirPath() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to locate user cache directory: %w", err)
	}
	return filepath.Join(cacheDir, appName), nil
}

func verifyDirectoryPermissions(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("cache path is not a directory: %s", path)
	}

	if info.Mode().Perm()&0077 != 0 {
		return fmt.Errorf("insecure permissions on cache directory: %#o", info.Mode().Perm())
	}

	return nil
}

func WritePortFile(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid port number: %d", port)
	}

	if err := EnsureAppCacheDir(); err != nil {
		return fmt.Errorf("cache directory validation failed: %w", err)
	}

	cacheDir, err := getAppCacheDirPath()
	if err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp(cacheDir, "port-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer cleanupTempFile(tmpFile)

	if err := performAtomicWrite(tmpFile, port); err != nil {
		return err
	}

	return finalizePortFile(tmpFile, cacheDir)
}

func performAtomicWrite(tmpFile *os.File, port int) error {
	if err := syscall.Flock(int(tmpFile.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("file lock failed: %w", err)
	}

	if _, err := fmt.Fprintf(tmpFile, "%d", port); err != nil {
		return fmt.Errorf("port write failed: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("file sync failed: %w", err)
	}

	return nil
}

func finalizePortFile(tmpFile *os.File, cacheDir string) error {
	finalPath := filepath.Join(cacheDir, portFileName)
	if err := os.Rename(tmpFile.Name(), finalPath); err != nil {
		return fmt.Errorf("atomic rename failed: %w", err)
	}

	// Ensure final file has correct permissions
	return os.Chmod(finalPath, filePerm)
}

func cleanupTempFile(tmpFile *os.File) {
	tmpFile.Close()
	os.Remove(tmpFile.Name())
}

func ReadPortFile() (int, error) {
	if err := EnsureAppCacheDir(); err != nil {
		return 0, fmt.Errorf("cache directory validation failed: %w", err)
	}

	cacheDir, err := getAppCacheDirPath()
	if err != nil {
		return 0, err
	}

	file, err := os.Open(filepath.Join(cacheDir, portFileName))
	if err != nil {
		return 0, fmt.Errorf("failed to open port file: %w", err)
	}
	defer file.Close()

	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_SH); err != nil {
		return 0, fmt.Errorf("file lock failed: %w", err)
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return 0, fmt.Errorf("read failed: %w", err)
	}

	return validatePortData(data)
}

func validatePortData(data []byte) (int, error) {
	portStr := string(bytes.TrimSpace(data))
	if portStr == "" {
		return 0, fmt.Errorf("empty port file")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, fmt.Errorf("invalid port format: %w", err)
	}

	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("port %d out of valid range", port)
	}

	return port, nil
}

func DeletePortFile() error {
	cacheDir, err := getAppCacheDirPath()
	if err != nil {
		return err
	}

	portPath := filepath.Join(cacheDir, portFileName)
	if err := os.Remove(portPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("delete failed: %w", err)
	}
	return nil
}
