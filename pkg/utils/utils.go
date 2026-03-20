package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EnsureDataDirExists creates the datadir and necessary subdirectories if they don't exist
func EnsureDataDirExists(datadir string) error {
	// Create main datadir
	if err := os.MkdirAll(datadir, 0700); err != nil {
		return fmt.Errorf("failed to create datadir %s: %v", datadir, err)
	}

	// Create logs subdirectory
	logsDir := filepath.Join(datadir, "logs")
	if err := os.MkdirAll(logsDir, 0700); err != nil {
		return fmt.Errorf("failed to create logs directory %s: %v", logsDir, err)
	}

	return nil
}

// FormatDCRAtoms formats an atom-denominated amount as a DCR decimal string.
func FormatDCRAtoms(atoms int64) string {
	sign := ""
	if atoms < 0 {
		sign = "-"
		atoms = -atoms
	}
	whole := atoms / 1e8
	frac := atoms % 1e8
	if frac == 0 {
		return fmt.Sprintf("%s%d", sign, whole)
	}
	fracStr := fmt.Sprintf("%08d", frac)
	fracStr = strings.TrimRight(fracStr, "0")
	return fmt.Sprintf("%s%d.%s", sign, whole, fracStr)
}
