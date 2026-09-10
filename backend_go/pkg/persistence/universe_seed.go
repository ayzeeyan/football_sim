package persistence

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const legacyUniverseSeed int64 = 20260907

// UniverseSeedPath returns the sidecar path that owns deterministic simulation
// identity for a career. The sidecar keeps the seed stable without changing
// the versioned career JSON schema, preserving older-save compatibility.
func UniverseSeedPath(savePath string) string {
	if savePath == "" {
		savePath = SavePath()
	}
	return savePath + ".seed"
}

// LoadOrCreateUniverseSeed restores a persisted universe seed or creates one
// exactly once for a new career. Existing legacy careers without a seed use a
// stable compatibility seed rather than inventing a new value on each launch.
func LoadOrCreateUniverseSeed(savePath string) (int64, error) {
	if savePath == "" {
		savePath = SavePath()
	}
	seedPath := UniverseSeedPath(savePath)
	if data, err := os.ReadFile(seedPath); err == nil {
		seed, parseErr := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
		if parseErr != nil || seed == 0 {
			if parseErr == nil {
				parseErr = fmt.Errorf("seed must be non-zero")
			}
			return 0, fmt.Errorf("invalid universe seed file %s: %w", seedPath, parseErr)
		}
		return seed, nil
	} else if !os.IsNotExist(err) {
		return 0, fmt.Errorf("failed to read universe seed file %s: %w", seedPath, err)
	}

	seed := legacyUniverseSeed
	if _, err := os.Stat(savePath); os.IsNotExist(err) {
		var raw [8]byte
		if _, err := rand.Read(raw[:]); err != nil {
			return 0, fmt.Errorf("failed to generate universe seed: %w", err)
		}
		seed = int64(binary.LittleEndian.Uint64(raw[:]) & 0x7fffffffffffffff)
		if seed == 0 {
			seed = 1
		}
	} else if err != nil {
		return 0, fmt.Errorf("failed to inspect career save while selecting universe seed: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(seedPath), 0755); err != nil {
		return 0, fmt.Errorf("failed to create universe seed directory: %w", err)
	}
	if err := writeAtomic(seedPath, []byte(strconv.FormatInt(seed, 10)+"\n")); err != nil {
		return 0, fmt.Errorf("failed to persist universe seed: %w", err)
	}
	return seed, nil
}
