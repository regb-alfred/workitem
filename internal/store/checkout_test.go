package store_test

import (
	"testing"
	"time"

	"github.com/regb/workitem/internal/model"
	"github.com/regb/workitem/internal/store"
)

func TestNormalizeManifestRejectsMalformedClaims(t *testing.T) {
	changes := map[string]func(*model.Manifest){
		"missing identity":           func(m *model.Manifest) { m.Checkout.Claim.ID = "" },
		"unsafe identity":            func(m *model.Manifest) { m.Checkout.Claim.ID = "../claim" },
		"missing generation":         func(m *model.Manifest) { m.Checkout.Claim.Generation = 0 },
		"missing durable generation": func(m *model.Manifest) { m.Checkout.ClaimGeneration = 0 },
		"generation mismatch":        func(m *model.Manifest) { m.Checkout.Claim.Generation = 2 },
		"relative path":              func(m *model.Manifest) { m.Checkout.Claim.Path = "workspace" },
		"unclean path":               func(m *model.Manifest) { m.Checkout.Claim.Path = "/workspace/../other" },
		"executor mismatch":          func(m *model.Manifest) { m.Checkout.Claim.Executor = "other" },
		"home branch intent": func(m *model.Manifest) {
			m.Checkout.Specification.Kind = model.WorkspaceKindRepositoryHome
			m.Checkout.Specification.Branch = "main"
		},
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			checkout := model.NewCheckout(model.WorkspaceKindManagedSlot, model.ExecutorLocal, "wi/example")
			if _, err := checkout.AssignNewClaim(model.ExecutorLocal, "repo", "slot-0001", "/workspace"); err != nil {
				t.Fatal(err)
			}
			m := model.NewManifest("01M17E1HRA465ERBR8H5Z4F68E", "example", "Example", nil, false, time.Now().UTC(), model.Repository{RootAtCreation: "/repository"}, checkout)
			change(&m)
			if _, err := store.NormalizeManifest(m); err == nil {
				t.Fatal("accepted malformed checkout")
			}
		})
	}
}
