package util

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GetExeDir returns the directory of the current executable.
func GetExeDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}

// ReadYesNo reads a yes/no response from the reader.
// Returns true for "Y" or empty input (default).
func ReadYesNo(r *bufio.Reader) bool {
	txt, _ := r.ReadString('\n')
	s := strings.ToUpper(strings.TrimSpace(txt))
	return s == "Y" || s == ""
}

// ReadTrimmed reads a trimmed string line from the reader.
func ReadTrimmed(r *bufio.Reader) string {
	txt, _ := r.ReadString('\n')
	return strings.TrimSpace(txt)
}

// CheckDocker checks if Docker is installed and running.
func CheckDocker() bool {
	_, err := exec.LookPath("docker")
	if err != nil {
		return false
	}
	err = exec.Command("docker", "info").Run()
	return err == nil
}

// EnsureDir creates a directory if it doesn't exist.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// FileExists checks if a file exists.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsDirectory checks if a path is a directory.
func IsDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
