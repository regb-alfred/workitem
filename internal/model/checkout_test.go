package model_test

import (
	"encoding/json"
	"testing"

	"github.com/regb/workitem/internal/model"
)

func TestCheckoutClaimGenerationSurvivesRelease(t *testing.T) {
	c := model.NewCheckout(model.WorkspaceKindManagedSlot, model.ExecutorLocal, "wi/example")
	first, err := c.AssignNewClaim(model.ExecutorLocal, "repo", "slot-0001", "/workspace")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.AssignNewClaim(model.ExecutorLocal, "repo", "slot-0001", "/workspace"); err == nil {
		t.Fatal("replaced active claim")
	}
	c.ClearClaim()
	if c.Present() || !c.EverMaterialized() || c.Specification.Branch != "wi/example" {
		t.Fatalf("released checkout: %+v", c)
	}
	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	var restored model.Checkout
	if err := json.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}
	next, err := restored.AssignNewClaim(model.ExecutorLocal, "repo", "slot-0001", "/workspace")
	if err != nil {
		t.Fatal(err)
	}
	if next.ID == first.ID || next.Generation != first.Generation+1 {
		t.Fatalf("claims: %+v %+v", first, next)
	}
}

func TestCheckoutClaimGenerationSafety(t *testing.T) {
	c := model.NewCheckout(model.WorkspaceKindManagedSlot, model.ExecutorLocal, "wi/example")
	if err := c.AcceptClaim(model.WorkspaceClaim{ID: "claim-2", Generation: 2}); err == nil {
		t.Fatal("accepted skipped generation")
	}
	c.ClaimGeneration = ^uint64(0)
	if _, err := c.AssignNewClaim(model.ExecutorLocal, "repo", "slot", "/workspace"); err == nil {
		t.Fatal("generation overflow accepted")
	}
	if err := c.AcceptClaim(model.WorkspaceClaim{ID: "claim-0", Generation: 0}); err == nil {
		t.Fatal("generation wraparound accepted")
	}
	if c.Claim != nil {
		t.Fatal("failed assignment changed checkout")
	}
}

func TestCheckoutRejectsObsoleteAndMalformedJSON(t *testing.T) {
	for _, raw := range []string{`{"kind":"managed-slot","branch":"wi/example","path":"/workspace"}`, `{"specification":null}`, `{"specification":{"kind":"managed-slot"},"path":"/workspace"}`, `{"specification":{},"claim_generation":-1}`, `{"specification":{},"claim":[]}`} {
		var c model.Checkout
		if err := json.Unmarshal([]byte(raw), &c); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
}

func TestRepositoryHomeHasNoBranchIntent(t *testing.T) {
	c := model.NewCheckout(model.WorkspaceKindRepositoryHome, model.ExecutorLocal, "old-branch")
	if c.Specification.Branch != "" {
		t.Fatalf("home branch intent: %q", c.Specification.Branch)
	}
}
