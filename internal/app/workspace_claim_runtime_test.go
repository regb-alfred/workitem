package app

import (
	"context"
	"strings"
	"testing"

	"github.com/regb/workitem/internal/model"
)

func TestRuntimeCannotResumeAgainstReplacementWorkspaceClaim(t *testing.T) {
	for _, claim := range []*model.WorkspaceClaim{nil, {ID: "claim-2", Generation: 2, Executor: model.ExecutorLocal, Path: "/workspace"}, {ID: "claim-1", Generation: 2, Executor: model.ExecutorLocal, Path: "/workspace"}} {
		manifest := model.Manifest{ID: "item-1", Checkout: model.Checkout{Specification: model.CheckoutSpecification{Executor: model.ExecutorLocal}, ClaimGeneration: 2, Claim: claim}}
		runtime := model.AgentRuntime{ID: "runtime-1", WorkspaceClaimID: "claim-1", WorkspaceClaimGeneration: 1}
		if _, err := (&App{}).validateRuntimeCheckout(context.Background(), manifest, runtime); err == nil || !strings.Contains(err.Error(), "superseded workspace claim") {
			t.Fatalf("claim=%+v err=%v", claim, err)
		}
	}
}
