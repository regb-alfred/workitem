package git_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	gitpkg "github.com/regb/workitem/internal/git"
	"github.com/regb/workitem/internal/model"
)

func TestDirtySnapshotPreservesIndexWorkingFilesDeletionsAndSymlinks(t *testing.T) {
	source := setupRepo(t)
	write := func(path string, data []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(source, path), data, 0755); err != nil {
			t.Fatal(err)
		}
	}
	write("binary", []byte{0, 1, 2, 3})
	write("deleted", []byte("delete me\n"))
	write("renamed", []byte("rename me\n"))
	write(".gitignore", []byte("ignored\n"))
	if err := os.Symlink("README.md", filepath.Join(source, "tracked-link")); err != nil {
		t.Fatal(err)
	}
	runGit(t, source, "add", ".")
	runGit(t, source, "commit", "-m", "snapshot base")
	head := runGitOutput(t, source, "rev-parse", "HEAD")
	if err := os.Remove(filepath.Join(source, "tracked-link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../outside-is-not-read", filepath.Join(source, "tracked-link")); err != nil {
		t.Fatal(err)
	}
	write("README.md", []byte("staged version\n"))
	runGit(t, source, "add", "README.md")
	write("README.md", []byte("unstaged version\n"))
	write("binary", []byte{3, 2, 1, 0})
	runGit(t, source, "add", "binary")
	write("binary", []byte{9, 8, 7, 6})
	if err := os.Remove(filepath.Join(source, "deleted")); err != nil {
		t.Fatal(err)
	}
	runGit(t, source, "mv", "renamed", "new-name")
	write("new-name", []byte("modified after rename\n"))
	write("untracked\nfile", []byte("new file\n"))
	write("ignored", []byte("do not copy\n"))
	if err := os.Symlink("README.md", filepath.Join(source, "untracked-link")); err != nil {
		t.Fatal(err)
	}
	client := gitpkg.New("git")
	snap, err := client.CaptureDirtySnapshot(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	before := runGitOutput(t, source, "status", "--porcelain=v1", "--untracked-files=all")
	root := filepath.Join(t.TempDir(), "owned.git")
	branch := runGitOutput(t, source, "branch", "--show-current")
	if err := client.ImportBare(context.Background(), gitpkg.BareImportOptions{Source: source, Destination: root, IntegrationBranch: branch}); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "slot")
	if err := client.WorktreeAdd(context.Background(), model.WorktreeAddOptions{RepoRoot: root, Path: destination, Branch: "wi/task", NewBranch: true, StartPoint: head}); err != nil {
		t.Fatal(err)
	}
	if err := client.RestoreDirtySnapshot(context.Background(), destination, snap); err != nil {
		t.Fatal(err)
	}
	restored, err := client.CaptureDirtySnapshot(context.Background(), destination)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(snap, restored) {
		t.Fatalf("snapshot changed during restore")
	}
	if got := runGitOutput(t, source, "status", "--porcelain=v1", "--untracked-files=all"); got != before {
		t.Fatalf("source was changed: %q != %q", got, before)
	}
	if _, err := os.Stat(filepath.Join(destination, "ignored")); !os.IsNotExist(err) {
		t.Fatalf("copied ignored file: %v", err)
	}
	if err := client.VerifyDirtySnapshot(context.Background(), source, snap); err != nil {
		t.Fatal(err)
	}
	write("README.md", []byte("changed after snapshot\n"))
	if err := client.VerifyDirtySnapshot(context.Background(), source, snap); err == nil {
		t.Fatal("failed to detect concurrent source changes")
	}
}

func TestDirtySnapshotIgnoresInheritedRepositoryOverrides(t *testing.T) {
	source := setupRepo(t)
	branch := runGitOutput(t, source, "branch", "--show-current")
	head := runGitOutput(t, source, "rev-parse", "HEAD")
	if err := os.WriteFile(filepath.Join(source, "README.md"), []byte("dirty source\n"), 0600); err != nil {
		t.Fatal(err)
	}
	client := gitpkg.New("git")
	snapshot, err := client.CaptureDirtySnapshot(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	bare := filepath.Join(t.TempDir(), "owned.git")
	if err := client.ImportBare(context.Background(), gitpkg.BareImportOptions{Source: source, Destination: bare, IntegrationBranch: branch}); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "slot")
	if err := client.WorktreeAdd(context.Background(), model.WorktreeAddOptions{RepoRoot: bare, Path: destination, Branch: "wi/task", NewBranch: true, StartPoint: head}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_DIR", filepath.Join(source, ".git"))
	t.Setenv("GIT_WORK_TREE", source)
	t.Setenv("GIT_INDEX_FILE", filepath.Join(source, ".git", "index"))
	if err := client.RestoreDirtySnapshot(context.Background(), destination, snapshot); err != nil {
		t.Fatal(err)
	}
	if err := client.VerifyDirtySnapshot(context.Background(), source, snapshot); err != nil {
		t.Fatal(err)
	}
	if err := client.VerifyDirtySnapshot(context.Background(), destination, snapshot); err != nil {
		t.Fatal(err)
	}
}

func TestDirtySnapshotRefusesSubdirectories(t *testing.T) {
	source := setupRepo(t)
	subdirectory := filepath.Join(source, "subdirectory")
	if err := os.Mkdir(subdirectory, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := gitpkg.New("git").CaptureDirtySnapshot(context.Background(), subdirectory); err == nil {
		t.Fatal("captured repo-relative paths using a subdirectory root")
	}
}

func TestDirtySnapshotRejectsUnsafeDestinations(t *testing.T) {
	for _, name := range []string{"../outside", "/absolute", ".git/config", "nested/../outside"} {
		t.Run(name, func(t *testing.T) {
			destination := setupRepo(t)
			head := runGitOutput(t, destination, "rev-parse", "HEAD")
			snap := gitpkg.DirtySnapshot{Head: head, Files: []gitpkg.DirtyFile{{Path: name, Mode: 0600, Data: []byte("unsafe")}}}
			if err := gitpkg.New("git").RestoreDirtySnapshot(context.Background(), destination, snap); err == nil {
				t.Fatal("accepted unsafe path")
			}
		})
	}
	destination := setupRepo(t)
	head := runGitOutput(t, destination, "rev-parse", "HEAD")
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(destination, "ignored-parent")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, ".git", "info", "exclude"), []byte("ignored-parent\n"), 0600); err != nil {
		t.Fatal(err)
	}
	snap := gitpkg.DirtySnapshot{Head: head, Files: []gitpkg.DirtyFile{{Path: "ignored-parent/file", Mode: 0600, Data: []byte("unsafe")}}}
	if err := gitpkg.New("git").RestoreDirtySnapshot(context.Background(), destination, snap); err == nil {
		t.Fatal("accepted symlink parent")
	}
	if _, err := os.Stat(filepath.Join(outside, "file")); !os.IsNotExist(err) {
		t.Fatalf("escaped destination: %v", err)
	}
}

func TestDirtySnapshotRefusesDirtyDestinationAndIgnoredFileCollision(t *testing.T) {
	destination := setupRepo(t)
	head := runGitOutput(t, destination, "rev-parse", "HEAD")
	snap := gitpkg.DirtySnapshot{Head: head, Files: []gitpkg.DirtyFile{{Path: "extra", Mode: 0600, Data: []byte("copied")}}}
	if err := os.WriteFile(filepath.Join(destination, "README.md"), []byte("existing changes"), 0600); err != nil {
		t.Fatal(err)
	}
	client := gitpkg.New("git")
	if err := client.RestoreDirtySnapshot(context.Background(), destination, snap); err == nil {
		t.Fatal("overwrote dirty destination")
	}
	runGit(t, destination, "restore", "README.md")
	if err := os.WriteFile(filepath.Join(destination, ".git", "info", "exclude"), []byte("extra\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "extra"), []byte("ignored existing data"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := client.RestoreDirtySnapshot(context.Background(), destination, snap); err == nil {
		t.Fatal("overwrote ignored file")
	}
}
