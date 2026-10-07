package workspace

import (
	"context"
	"testing"
	"time"

	"github.com/regb/workitem/internal/app/contract"
	"github.com/regb/workitem/internal/model"
)

type branchPolicyGit struct {
	fakeGit
	branch string
}

func (g branchPolicyGit) CurrentBranch(context.Context, string) (string, error) { return g.branch, nil }
func (branchPolicyGit) StatusPorcelain(context.Context, string) (string, error) { return "", nil }

func TestBranchPolicyAccessAndRelease(t *testing.T) {
	for _, kind := range []string{model.WorkspaceKindRepositoryHome, model.WorkspaceKindManagedSlot} {
		for _, branch := range []string{"main", "experiment", ""} {
			t.Run(kind+"/"+branch, func(t *testing.T) {
				path := t.TempDir()
				m := model.Manifest{ID: "item-1", State: model.StateWorking, Checkout: model.Checkout{Kind: kind, Branch: "master", Path: &path}}
				resolve := func(context.Context, contract.ResolveOptions) (model.Manifest, error) { return m, nil }
				s := New(&recordingStore{}, branchPolicyGit{branch: branch}, nil, nil, resolve, nil, time.Now)
				ensured, _, err := s.EnsureCheckout(context.Background(), contract.ResolveOptions{}, m, true)
				if err != nil || ensured.Checkout.Branch != "master" {
					t.Fatalf("ensure=%+v err=%v", ensured, err)
				}
				if _, err := s.CheckoutCreate(context.Background(), contract.ResolveOptions{}); err != nil {
					t.Fatal(err)
				}
				status, err := s.Status(context.Background(), contract.ResolveOptions{})
				if err != nil {
					t.Fatal(err)
				}
				strictErr := s.ensureCheckoutBranch(context.Background(), m)
				if kind == model.WorkspaceKindRepositoryHome {
					if strictErr != nil || status.Git.ExpectedBranch != "" || status.Git.BranchMismatch {
						t.Fatalf("status=%+v err=%v", status, strictErr)
					}
					released, err := s.Release(context.Background(), contract.ResolveOptions{}, false)
					if err != nil || released.Manifest.Checkout.Present() {
						t.Fatalf("release=%+v err=%v", released, err)
					}
				} else if strictErr == nil || !status.Git.BranchMismatch {
					t.Fatalf("managed release must remain strict: status=%+v err=%v", status, strictErr)
				}
			})
		}
	}
}
