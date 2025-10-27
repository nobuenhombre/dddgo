package helpers

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/nobuenhombre/suikat/pkg/ge"
)

// FindProjectRoot attempts to locate the root directory of the current Go project.
// It traverses up the directory tree starting from the caller's file location
// until it finds a directory containing a go.mod file.
//
// Returns:
//   - The absolute path to the project root directory if found
//   - An empty string if the project root cannot be located
func FindProjectRoot() (string, error) {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		return "", ge.New("cannot find project root")
	}

	current := filepath.Dir(filename)

	for {
		_, err := os.Stat(filepath.Join(current, "go.mod"))
		if err == nil {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}

		current = parent
	}

	return "", ge.New("cannot find project root")
}
