package primaryagent

import (
	"context"
	"errors"
	"testing"

	"github.com/regb/workitem/internal/model"
)

type branchObservationGit struct {
	branch, head, dirty string
	err                 error
}

func (g branchObservationGit) Head(context.Context, string) (string, error) { return g.head, g.err }
func (g branchObservationGit) CurrentBranch(context.Context, string) (string, error) {
	return g.branch, g.err
}
func (g branchObservationGit) StatusPorcelain(context.Context, string) (string, error) {
	return g.dirty, g.err
}

func TestWorktreeBranchPolicy(t *testing.T) {
	for _, tc := range []struct {
		name, kind, branch, head, dirty, status string
		mismatch                                bool
		err                                     error
	}{
		{name: "home rename", kind: model.WorkspaceKindRepositoryHome, branch: "main", head: "base", status: "clean"},
		{name: "home temporary branch", kind: model.WorkspaceKindRepositoryHome, branch: "experiment", head: "new", status: "changed"},
		{name: "home detached", kind: model.WorkspaceKindRepositoryHome, head: "base", status: "clean"},
		{name: "home dirty", kind: model.WorkspaceKindRepositoryHome, branch: "main", head: "base", dirty: " M file", status: "changed"},
		{name: "managed divergence", kind: model.WorkspaceKindManagedSlot, branch: "experiment", head: "base", status: "changed", mismatch: true},
		{name: "managed detached", kind: model.WorkspaceKindManagedSlot, head: "base", status: "changed", mismatch: true},
		{name: "inspection failure", kind: model.WorkspaceKindRepositoryHome, status: "problem", err: errors.New("git failed")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := t.TempDir()
			m := model.Manifest{Repository: model.Repository{CreatedFromCommit: "base"}, Checkout: model.Checkout{Specification: model.CheckoutSpecification{Kind: tc.kind, Branch: "master"}, ClaimGeneration: 1, Claim: &model.WorkspaceClaim{ID: "test-claim", Generation: 1, Executor: model.ExecutorLocal, Path: path}}}
			s := New(nil, nil, branchObservationGit{tc.branch, tc.head, tc.dirty, tc.err}, nil, nil, nil, nil, nil)
			got, warnings := s.inspectWorktree(context.Background(), m)
			if got.Status != tc.status || got.BranchMismatch != tc.mismatch || got.CurrentBranch != tc.branch {
				t.Fatalf("status=%+v warnings=%v", got, warnings)
			}
			if tc.kind == model.WorkspaceKindRepositoryHome && (got.ExpectedBranch != "" || got.BranchMatches) {
				t.Fatalf("home has branch constraint: %+v", got)
			}
			if tc.err == nil && !tc.mismatch && len(warnings) != 0 {
				t.Fatalf("warnings=%v", warnings)
			}
		})
	}
}
