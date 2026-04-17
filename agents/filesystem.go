//nolint:forbidigo // thin wrapper over os.* is the point
package agents

import (
	"os"
)

// FileSystem provides abstraction for file system operations.
// This interface enables testing by allowing mock implementations.
type FileSystem interface {
	// WorkingDir returns the current working directory.
	WorkingDir() (string, error)

	// MkdirAll creates a directory along with any necessary parents.
	MkdirAll(path string, perm os.FileMode) error

	// Mkdir creates a directory.
	Mkdir(path string, perm os.FileMode) error

	// RemoveAll removes a path and any children it contains.
	RemoveAll(path string) error

	// WriteFile writes data to a file.
	WriteFile(path string, data []byte, perm os.FileMode) error

	// Stat returns file information.
	Stat(path string) (os.FileInfo, error)
}

// OSFileSystem implements FileSystem using the os package.
type OSFileSystem struct{}

// WorkingDir returns the current working directory.
func (fs *OSFileSystem) WorkingDir() (string, error) {
	return os.Getwd()
}

// MkdirAll creates a directory along with any necessary parents.
func (fs *OSFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// Mkdir creates a directory.
func (fs *OSFileSystem) Mkdir(path string, perm os.FileMode) error {
	return os.Mkdir(path, perm)
}

// RemoveAll removes a path and any children it contains.
func (fs *OSFileSystem) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

// WriteFile writes data to a file.
func (fs *OSFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

// Stat returns file information.
func (fs *OSFileSystem) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}
