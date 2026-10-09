package git_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gitpkg "github.com/regb/workitem/internal/git"
	"github.com/regb/workitem/internal/model"
)

func TestImportBareIsIndependentAndLeavesIntegrationBranchUncheckedOut(t *testing.T) {
	source := setupRepo(t)
	branch := runGitOutput(t, source, "branch", "--show-current")
	head := runGitOutput(t, source, "rev-parse", "HEAD")
	runGit(t, source, "branch", "wi/local-only")
	if err := os.WriteFile(filepath.Join(source, "dirty.txt"), []byte("not imported\n"), 0600); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "owned.git")
	client := gitpkg.New("git")
	if err := client.ImportBare(context.Background(), gitpkg.BareImportOptions{Source: source, Destination: destination, IntegrationBranch: branch, RequiredCommits: []string{head}}); err != nil {
		t.Fatal(err)
	}
	if err := client.ValidateOwnedBare(context.Background(), destination, branch); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(destination, "objects", "info", "alternates")); !os.IsNotExist(err) {
		t.Fatalf("alternates: %v", err)
	}
	if got := runGitOutput(t, destination, "rev-parse", "refs/heads/wi/local-only"); got != head {
		t.Fatalf("local-only branch = %s", got)
	}
	if got := runGitOutput(t, destination, "rev-parse", "refs/wi/pins/"+head); got != head {
		t.Fatalf("commit pin = %s", got)
	}
	list, err := client.ListWorktrees(context.Background(), destination)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || !list[0].Bare {
		t.Fatalf("worktrees = %+v", list)
	}
	if err := os.Rename(source, source+"-moved"); err != nil {
		t.Fatal(err)
	}
	if got := runGitOutput(t, destination, "rev-parse", "HEAD"); got != head {
		t.Fatalf("import depends on source: %s", got)
	}
	path := filepath.Join(t.TempDir(), "workspace")
	if err := client.WorktreeAdd(context.Background(), model.WorktreeAddOptions{RepoRoot: destination, Path: path, Branch: "wi/task", NewBranch: true, StartPoint: head}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(path, "dirty.txt")); !os.IsNotExist(err) {
		t.Fatalf("dirty files were implicitly transferred: %v", err)
	}
	if _, err := client.CurrentBranch(context.Background(), path); err != nil {
		t.Fatal(err)
	}
}

func TestImportedRepositoryFetchOnlyUpdatesRemoteTrackingRefs(t *testing.T) {
	upstream := setupRepo(t)
	branch := runGitOutput(t, upstream, "branch", "--show-current")
	old := runGitOutput(t, upstream, "rev-parse", "HEAD")
	destination := filepath.Join(t.TempDir(), "owned.git")
	client := gitpkg.New("git")
	if err := client.ImportBare(context.Background(), gitpkg.BareImportOptions{Source: upstream, Destination: destination, OriginURL: upstream, IntegrationBranch: branch}); err != nil {
		t.Fatal(err)
	}
	runGit(t, destination, "branch", "wi/local", old)
	runGit(t, upstream, "commit", "--allow-empty", "-m", "upstream advanced")
	next := runGitOutput(t, upstream, "rev-parse", "HEAD")
	runGit(t, destination, "fetch", "origin")
	if got := runGitOutput(t, destination, "rev-parse", "refs/heads/"+branch); got != old {
		t.Fatal("fetch overwrote integration branch")
	}
	if got := runGitOutput(t, destination, "rev-parse", "refs/heads/wi/local"); got != old {
		t.Fatal("fetch overwrote item branch")
	}
	if got := runGitOutput(t, destination, "rev-parse", "refs/remotes/origin/"+branch); got != next {
		t.Fatal("fetch failed to advance remote-tracking ref")
	}
}

func TestOwnedStoreRefusesCheckedOutIntegrationBranch(t *testing.T) {
	source := setupRepo(t)
	branch := runGitOutput(t, source, "branch", "--show-current")
	path := filepath.Join(t.TempDir(), "owned.git")
	client := gitpkg.New("git")
	if err := client.ImportBare(context.Background(), gitpkg.BareImportOptions{Source: source, Destination: path, IntegrationBranch: branch}); err != nil {
		t.Fatal(err)
	}
	checkout := filepath.Join(t.TempDir(), "integration")
	if err := client.WorktreeAdd(context.Background(), model.WorktreeAddOptions{RepoRoot: path, Path: checkout, Branch: branch}); err != nil {
		t.Fatal(err)
	}
	if err := client.ValidateOwnedBare(context.Background(), path, branch); err == nil || !strings.Contains(err.Error(), "checked out") {
		t.Fatalf("error = %v", err)
	}
}

func TestImportBareFailureDoesNotPublishAnIncompleteStore(t *testing.T) {
	source := setupRepo(t)
	destination := filepath.Join(t.TempDir(), "owned.git")
	err := gitpkg.New("git").ImportBare(context.Background(), gitpkg.BareImportOptions{Source: source, Destination: destination, IntegrationBranch: "missing-branch"})
	if err == nil {
		t.Fatal("missing integration branch was accepted")
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatalf("published incomplete store: %v", err)
	}
	entries, err := os.ReadDir(filepath.Dir(destination))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("left staging directories: %+v", entries)
	}
}

func TestValidateOwnedBareRejectsSymlinksAndAlternates(t *testing.T) {
	source := setupRepo(t)
	branch := runGitOutput(t, source, "branch", "--show-current")
	destination := filepath.Join(t.TempDir(), "owned.git")
	client := gitpkg.New("git")
	if err := client.ImportBare(context.Background(), gitpkg.BareImportOptions{Source: source, Destination: destination, IntegrationBranch: branch}); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "linked.git")
	if err := os.Symlink(destination, link); err != nil {
		t.Fatal(err)
	}
	if err := client.ValidateOwnedBare(context.Background(), link, branch); err == nil {
		t.Fatal("accepted symlink store")
	}
	if err := os.WriteFile(filepath.Join(destination, "objects", "info", "alternates"), []byte(filepath.Join(source, ".git", "objects")+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := client.ValidateOwnedBare(context.Background(), destination, branch); err == nil || !strings.Contains(err.Error(), "alternates") {
		t.Fatalf("error = %v", err)
	}
}
