package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// Nested untracked repositories are opaque to the enclosing Git index. Preserve
// their complete contents, including their own Git metadata, without traversing
// symlinks. Digest both endpoints and the source again after copying.
func directoryDigest(root string) (string, error) {
	hash := sha256.New()
	write := func(b []byte) {
		var n [8]byte
		binary.BigEndian.PutUint64(n[:], uint64(len(b)))
		hash.Write(n[:])
		hash.Write(b)
	}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		write([]byte(rel))
		write([]byte(fmt.Sprintf("%d", info.Mode())))
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			write([]byte(target))
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported opaque directory entry: %s", path)
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		contents := sha256.New()
		_, err = io.Copy(contents, f)
		closeErr := f.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		write(contents.Sum(nil))
		return nil
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
func copyDirectory(source, target, expected string) error {
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		return fmt.Errorf("opaque destination already exists: %s", target)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return err
	}
	cmd := exec.Command("cp", "-a", "--", source, target)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("copy opaque directory: %s: %w", out, err)
	}
	copied, err := directoryDigest(target)
	if err != nil {
		return err
	}
	if copied != expected {
		return fmt.Errorf("opaque copy changed: %s", target)
	}
	original, err := directoryDigest(source)
	if err != nil {
		return err
	}
	if original != expected {
		return fmt.Errorf("opaque source changed during copy: %s", source)
	}
	return nil
}
