// Package repo finds the checkout root so commands can run from that directory.
package repo

import (
	"fmt"
	"os"
	"path/filepath"
)

// Root walks up from the working directory until it finds go.mod.
func Root() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from %s; run from the repository root", dir)
		}
		dir = parent
	}
}
