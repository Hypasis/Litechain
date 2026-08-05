package version

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestAutoUpdateManagerStagingAndSwapping(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "litechain-upgrade-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager, err := NewAutoUpdateManager(tempDir)
	if err != nil {
		t.Fatalf("failed to create AutoUpdateManager: %v", err)
	}

	// Mock binary content
	dummyBinaryContent := []byte("#!/bin/sh\necho 'Litechain v2.0.0'")
	hasher := sha256.New()
	hasher.Write(dummyBinaryContent)
	expectedHash := hex.EncodeToString(hasher.Sum(nil))

	releaseMeta := &NodeReleaseMetadata{
		Version:        "2.0.0",
		Type:           ReleaseTypeHardFork,
		TargetHeight:   500,
		BinaryName:     "lightchain",
		SHA256Checksum: expectedHash,
	}

	// 1. Stage Release
	err = manager.StageRelease(releaseMeta, dummyBinaryContent)
	if err != nil {
		t.Fatalf("StageRelease failed: %v", err)
	}

	// 2. Apply Upgrade (Symlink Swap)
	swappedPath, err := manager.ApplyUpgrade("2.0.0")
	if err != nil {
		t.Fatalf("ApplyUpgrade failed: %v", err)
	}

	// 3. Verify symlink points to target file
	stagedFile := filepath.Join(tempDir, "binaries", "lightchain-2.0.0")
	if swappedPath != stagedFile {
		t.Errorf("expected swapped path %s, got %s", stagedFile, swappedPath)
	}

	symlinkTarget, err := os.Readlink(manager.currentSym)
	if err != nil || symlinkTarget != stagedFile {
		t.Errorf("symlink target mismatch: expected %s, got %s", stagedFile, symlinkTarget)
	}
}

func TestChecksumVerificationFailure(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "litechain-checksum-test-*")
	defer os.RemoveAll(tempDir)

	manager, _ := NewAutoUpdateManager(tempDir)

	dummyBinaryContent := []byte("corrupted_binary_data")
	releaseMeta := &NodeReleaseMetadata{
		Version:        "2.0.1",
		Type:           ReleaseTypeHotfix,
		SHA256Checksum: "invalid_expected_hash_value_1234567890abcdef",
	}

	err := manager.StageRelease(releaseMeta, dummyBinaryContent)
	if err == nil {
		t.Fatalf("expected StageRelease to fail due to checksum mismatch")
	}
}
