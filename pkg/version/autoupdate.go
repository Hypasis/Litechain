package version

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// ReleaseType classifies the impact of a node release
type ReleaseType string

const (
	// ReleaseTypeHotfix represents a non-breaking patch or performance update. Optional immediate application.
	ReleaseTypeHotfix ReleaseType = "HOTFIX"

	// ReleaseTypeHardFork represents a breaking state transition / protocol change requiring height-bound upgrade.
	ReleaseTypeHardFork ReleaseType = "HARD_FORK"
)

// NodeReleaseMetadata defines metadata for a published binary release
type NodeReleaseMetadata struct {
	Version        string      `json:"version"`
	Type           ReleaseType `json:"type"`
	TargetHeight   uint64      `json:"targetHeight"` // Set for HARD_FORK releases
	BinaryName     string      `json:"binaryName"`
	SHA256Checksum string      `json:"sha256Checksum"`
	DownloadURL    string      `json:"downloadUrl"`
	ReleaseNotes   string      `json:"releaseNotes"`
}

// AutoUpdateManager handles zero-downtime binary staging and symlink swapping for node upgrades
type AutoUpdateManager struct {
	mu           sync.RWMutex
	baseDir      string
	binariesDir  string
	currentSym   string
	pendingRelease *NodeReleaseMetadata
}

// NewAutoUpdateManager initializes the node binary upgrade manager
func NewAutoUpdateManager(baseDir string) (*AutoUpdateManager, error) {
	binariesDir := filepath.Join(baseDir, "binaries")
	if err := os.MkdirAll(binariesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create binaries directory: %w", err)
	}

	currentSym := filepath.Join(baseDir, "current")

	return &AutoUpdateManager{
		baseDir:     baseDir,
		binariesDir: binariesDir,
		currentSym:  currentSym,
	}, nil
}

// StageRelease verifies binary checksum and stores the binary in the binaries directory
func (m *AutoUpdateManager) StageRelease(meta *NodeReleaseMetadata, binaryData []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Verify SHA256 Checksum
	hasher := sha256.New()
	hasher.Write(binaryData)
	computedHash := hex.EncodeToString(hasher.Sum(nil))

	if meta.SHA256Checksum != "" && computedHash != meta.SHA256Checksum {
		return fmt.Errorf("checksum mismatch for release %s: expected %s, got %s",
			meta.Version, meta.SHA256Checksum, computedHash)
	}

	// Write staged binary
	targetFile := filepath.Join(m.binariesDir, fmt.Sprintf("lightchain-%s", meta.Version))
	if err := os.WriteFile(targetFile, binaryData, 0755); err != nil {
		return fmt.Errorf("failed to write staged binary: %w", err)
	}

	m.pendingRelease = meta
	return nil
}

// ApplyUpgrade swaps the 'current' executable symlink to point to the newly staged binary
func (m *AutoUpdateManager) ApplyUpgrade(versionStr string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	targetFile := filepath.Join(m.binariesDir, fmt.Sprintf("lightchain-%s", versionStr))
	if _, err := os.Stat(targetFile); os.IsNotExist(err) {
		return "", fmt.Errorf("staged binary for version %s does not exist", versionStr)
	}

	// Remove old symlink if exists
	_ = os.Remove(m.currentSym)

	// Create new symlink
	if err := os.Symlink(targetFile, m.currentSym); err != nil {
		return "", fmt.Errorf("failed to create symlink: %w", err)
	}

	return targetFile, nil
}

// VerifyFileChecksum calculates SHA256 of a local file
func VerifyFileChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
