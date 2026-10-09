package git

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

// DirtySnapshot is native migration content, never a daemon event or projection.
// It preserves index changes separately from working files, including deletions
// and symlinks. Ignored files are intentionally not included.
type DirtySnapshot struct {
	Head          string
	StagedPatch   []byte
	UnstagedPatch []byte
	Files         []DirtyFile
	// Opaque nested repositories require separately verified recursive copying.
	Directories []string
}

type DirtyFile struct {
	Path    string
	Tracked bool
	Deleted bool
	Mode    os.FileMode
	Data    []byte
}

const maxDirtySnapshotBytes = 256 << 20

func (c Client) snapshotOutput(ctx context.Context, dir string, input []byte, args ...string) ([]byte, error) {
	executable := c.Path
	if executable == "" {
		executable = "git"
	}
	command := exec.CommandContext(ctx, executable, append([]string{"-C", dir}, args...)...)
	command.Env = append(repositoryEnvironment(), "GIT_OPTIONAL_LOCKS=0")
	command.Stdin = bytes.NewReader(input)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), strings.TrimSpace(stderr.String()), err)
	}
	if stdout.Len() > maxDirtySnapshotBytes {
		return nil, fmt.Errorf("dirty snapshot exceeds %d bytes", maxDirtySnapshotBytes)
	}
	return stdout.Bytes(), nil
}

func snapshotPath(root, name string, createParents bool) (string, error) {
	relative := filepath.FromSlash(name)
	if name == "" || filepath.IsAbs(relative) || filepath.Clean(relative) != relative || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe dirty-file path %q", name)
	}
	parts := strings.Split(relative, string(filepath.Separator))
	for _, part := range parts {
		if strings.EqualFold(part, ".git") {
			return "", fmt.Errorf("dirty file must not contain Git metadata: %q", name)
		}
	}
	current := root
	for _, part := range parts[:len(parts)-1] {
		current = filepath.Join(current, part)
		if createParents {
			if err := os.Mkdir(current, 0755); err != nil && !os.IsExist(err) {
				return "", err
			}
		}
		info, err := os.Lstat(current)
		if err != nil {
			return "", err
		}
		if !info.IsDir() {
			return "", fmt.Errorf("dirty-file parent is not a directory: %s", current)
		}
	}
	return filepath.Join(root, relative), nil
}

func (c Client) CaptureDirtySnapshot(ctx context.Context, source string) (DirtySnapshot, error) {
	var snap DirtySnapshot
	root, err := c.snapshotOutput(ctx, source, nil, "rev-parse", "--show-toplevel")
	if err != nil {
		return snap, err
	}
	expectedRoot, err := filepath.EvalSymlinks(source)
	if err != nil {
		return snap, err
	}
	expectedRoot, err = filepath.Abs(expectedRoot)
	if err != nil {
		return snap, err
	}
	actualRoot, err := filepath.EvalSymlinks(strings.TrimSpace(string(root)))
	if err != nil {
		return snap, err
	}
	if filepath.Clean(actualRoot) != expectedRoot {
		return snap, fmt.Errorf("snapshot source must be the checkout root, not a subdirectory")
	}
	head, err := c.snapshotOutput(ctx, source, nil, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return snap, err
	}
	snap.Head = strings.TrimSpace(string(head))
	if err := c.EnsureNoOperationInProgress(ctx, source, "snapshot checkout"); err != nil {
		return snap, err
	}
	unmerged, err := c.snapshotOutput(ctx, source, nil, "ls-files", "--unmerged", "-z")
	if err != nil {
		return snap, err
	}
	if len(unmerged) > 0 {
		return snap, fmt.Errorf("cannot migrate an unmerged index")
	}
	index, err := c.snapshotOutput(ctx, source, nil, "ls-files", "--stage", "-z")
	if err != nil {
		return snap, err
	}
	for _, entry := range bytes.Split(index, []byte{0}) {
		if bytes.HasPrefix(entry, []byte("160000 ")) {
			return snap, fmt.Errorf("submodule worktrees require a separate migration plan")
		}
	}
	diffFlags := []string{"--binary", "--full-index", "--no-ext-diff", "--no-textconv", "--no-renames"}
	snap.StagedPatch, err = c.snapshotOutput(ctx, source, nil, append([]string{"diff", "--cached"}, append(diffFlags, "HEAD", "--")...)...)
	if err != nil {
		return snap, err
	}
	snap.UnstagedPatch, err = c.snapshotOutput(ctx, source, nil, append([]string{"diff"}, append(diffFlags, "--")...)...)
	if err != nil {
		return snap, err
	}
	files := map[string]bool{}
	for _, args := range [][]string{{"diff", "--cached", "--name-only", "--no-renames", "-z", "HEAD", "--"}, {"diff", "--name-only", "--no-renames", "-z", "--"}} {
		names, err := c.snapshotOutput(ctx, source, nil, args...)
		if err != nil {
			return snap, err
		}
		for _, name := range bytes.Split(names, []byte{0}) {
			if len(name) > 0 {
				files[string(name)] = true
			}
		}
	}
	untracked, err := c.snapshotOutput(ctx, source, nil, "ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return snap, err
	}
	for _, name := range bytes.Split(untracked, []byte{0}) {
		if len(name) > 0 {
			if strings.HasSuffix(string(name), "/") {
				clean := strings.TrimSuffix(string(name), "/")
				path, err := snapshotPath(source, clean, false)
				if err != nil {
					return snap, err
				}
				info, err := os.Lstat(path)
				if err != nil {
					return snap, err
				}
				if !info.IsDir() {
					return snap, fmt.Errorf("opaque untracked path is not a directory: %s", clean)
				}
				snap.Directories = append(snap.Directories, clean)
				continue
			}
			files[string(name)] = false
		}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	size := len(snap.StagedPatch) + len(snap.UnstagedPatch)
	for _, name := range names {
		file := DirtyFile{Path: name, Tracked: files[name]}
		path, err := snapshotPath(source, name, false)
		// A tracked deletion may also remove the file's containing directory.
		if err != nil && !(file.Tracked && os.IsNotExist(err)) {
			return snap, err
		}
		if path == "" {
			path = filepath.Join(source, filepath.FromSlash(name))
		}
		info, err := os.Lstat(path)
		if os.IsNotExist(err) && file.Tracked {
			file.Deleted = true
			snap.Files = append(snap.Files, file)
			continue
		}
		if err != nil {
			return snap, err
		}
		file.Mode = info.Mode() & (os.ModeType | os.ModePerm)
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err != nil {
				return snap, err
			}
			file.Data = []byte(target)
		case info.Mode().IsRegular():
			if info.Size() > int64(maxDirtySnapshotBytes-size) {
				return snap, fmt.Errorf("dirty snapshot exceeds %d bytes", maxDirtySnapshotBytes)
			}
			opened, err := os.Open(path)
			if err != nil {
				return snap, err
			}
			actual, err := opened.Stat()
			if err != nil {
				_ = opened.Close()
				return snap, err
			}
			if !os.SameFile(info, actual) {
				_ = opened.Close()
				return snap, fmt.Errorf("dirty file changed while opening: %s", name)
			}
			file.Data, err = io.ReadAll(io.LimitReader(opened, int64(maxDirtySnapshotBytes-size)+1))
			closeErr := opened.Close()
			if err != nil {
				return snap, err
			}
			if closeErr != nil {
				return snap, closeErr
			}
		default:
			return snap, fmt.Errorf("unsupported dirty-file type: %s", name)
		}
		size += len(file.Data)
		if size > maxDirtySnapshotBytes {
			return snap, fmt.Errorf("dirty snapshot exceeds %d bytes", maxDirtySnapshotBytes)
		}
		snap.Files = append(snap.Files, file)
	}
	return snap, nil
}

// VerifyDirtySnapshot must run again after runtimes have stopped and before
// cutover. A successful earlier capture does not freeze the source checkout.
func (c Client) VerifyDirtySnapshot(ctx context.Context, source string, snap DirtySnapshot) error {
	current, err := c.CaptureDirtySnapshot(ctx, source)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(current, snap) {
		return fmt.Errorf("checkout changed since its migration snapshot: %s", source)
	}
	return nil
}

// RestoreDirtySnapshot only accepts a clean disposable destination at the exact
// captured commit. On failure, keep that provisional worktree for inspection;
// neither the source nor a canonical workspace claim is modified here.
func (c Client) RestoreDirtySnapshot(ctx context.Context, destination string, snap DirtySnapshot) error {
	if !objectIDPattern.MatchString(snap.Head) {
		return fmt.Errorf("invalid snapshot HEAD")
	}
	initial, err := c.CaptureDirtySnapshot(ctx, destination)
	if err != nil {
		return err
	}
	if initial.Head != snap.Head || len(initial.StagedPatch) != 0 || len(initial.UnstagedPatch) != 0 || len(initial.Files) != 0 {
		return fmt.Errorf("snapshot destination must be clean at commit %s", snap.Head)
	}
	seen := map[string]bool{}
	for _, file := range snap.Files {
		if seen[file.Path] {
			return fmt.Errorf("duplicate dirty-file path %q", file.Path)
		}
		seen[file.Path] = true
		path, err := snapshotPath(destination, file.Path, false)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if !file.Tracked && path != "" {
			if _, err := os.Lstat(path); err == nil {
				return fmt.Errorf("dirty-file destination already exists: %s", file.Path)
			} else if !os.IsNotExist(err) {
				return err
			}
		}
		if !file.Deleted && file.Mode.Type() != 0 && file.Mode.Type() != os.ModeSymlink {
			return fmt.Errorf("unsupported snapshot file type: %s", file.Path)
		}
	}
	if len(snap.StagedPatch) > 0 {
		if _, err := c.snapshotOutput(ctx, destination, snap.StagedPatch, "apply", "--index", "--binary", "--whitespace=nowarn", "-"); err != nil {
			return err
		}
	}
	if len(snap.UnstagedPatch) > 0 {
		if _, err := c.snapshotOutput(ctx, destination, snap.UnstagedPatch, "apply", "--binary", "--whitespace=nowarn", "-"); err != nil {
			return err
		}
	}
	for _, file := range snap.Files {
		path, err := snapshotPath(destination, file.Path, !file.Deleted)
		if file.Deleted {
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return err
			}
			info, err := os.Lstat(path)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return err
			}
			if info.IsDir() {
				return fmt.Errorf("refusing to remove directory %s", path)
			}
			if err := os.Remove(path); err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		if file.Mode&os.ModeSymlink != 0 {
			if file.Tracked {
				if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
					return err
				}
			}
			if err := os.Symlink(string(file.Data), path); err != nil {
				return err
			}
			continue
		}
		temp, err := os.CreateTemp(filepath.Dir(path), ".wi-migration-*")
		if err != nil {
			return err
		}
		name := temp.Name()
		err = temp.Chmod(file.Mode.Perm())
		if err == nil {
			_, err = temp.Write(file.Data)
		}
		if err == nil {
			err = temp.Sync()
		}
		closeErr := temp.Close()
		if err == nil {
			err = closeErr
		}
		if err == nil && !file.Tracked {
			if _, e := os.Lstat(path); e == nil {
				err = fmt.Errorf("dirty-file destination appeared: %s", path)
			} else if !os.IsNotExist(e) {
				err = e
			}
		}
		if err == nil {
			err = os.Rename(name, path)
		}
		_ = os.Remove(name)
		if err != nil {
			return err
		}
	}
	return c.VerifyDirtySnapshot(ctx, destination, snap)
}
