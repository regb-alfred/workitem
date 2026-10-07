package coordinator

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/regb/workitem/internal/model"
)

func TestProjectedWorktreeBranchPolicy(t *testing.T) {
	path := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", path}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s: %v", args, out, err)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-b", "main")
	git("-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "--allow-empty", "-m", "initial")
	base := git("rev-parse", "HEAD")
	observer := NewExternalObserver(nil, t.TempDir())
	for _, kind := range []string{model.WorkspaceKindRepositoryHome, model.WorkspaceKindManagedSlot} {
		for _, detached := range []bool{false, true} {
			if detached {
				git("checkout", "--detach", base)
			} else {
				git("checkout", "main")
			}
			m := model.Manifest{Repository: model.Repository{CreatedFromCommit: base}, Checkout: model.Checkout{Kind: kind, Branch: "master", Path: &path}}
			got := observer.observeWorktree(context.Background(), m)
			if kind == model.WorkspaceKindRepositoryHome {
				if got.Status != "clean" || got.ExpectedBranch != "" || got.BranchMismatch {
					t.Fatalf("home=%+v", got)
				}
			} else if got.Status != "changed" || !got.BranchMismatch || !got.HasChanges {
				t.Fatalf("managed=%+v", got)
			}
		}
	}
}
