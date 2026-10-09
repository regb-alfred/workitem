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
	for _, detached := range []bool{false, true} {
		if detached {
			git("checkout", "--detach", base)
		} else {
			git("checkout", "main")
		}
		m := model.Manifest{Repository: model.Repository{CreatedFromCommit: base}, Checkout: model.Checkout{Specification: model.CheckoutSpecification{Kind: model.WorkspaceKindManagedSlot, Branch: "master"}, ClaimGeneration: 1, Claim: &model.WorkspaceClaim{ID: "test-claim", Generation: 1, Executor: model.ExecutorLocal, Path: path}}}
		got := observer.observeWorktree(context.Background(), m)
		if got.Status != "changed" || !got.BranchMismatch || !got.HasChanges {
			t.Fatalf("managed=%+v", got)
		}
	}
}
