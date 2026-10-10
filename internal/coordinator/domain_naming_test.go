package coordinator

import (
	"context"
	"github.com/regb/workitem/internal/model"
	"github.com/regb/workitem/internal/store"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestNamingFinalizationKeepsIdentityAndAllocatesNamesAtomically(t *testing.T) {
	root := t.TempDir()
	native := store.New(root)
	if err := native.Ensure(); err != nil {
		t.Fatal(err)
	}
	source := model.Repository{RootAtCreation: "/repo", CreatedFromCommit: "original"}
	m := model.NewManifest(importTestID, "new-conversation", "New conversation", nil, false, time.Now(), source, model.NewCheckout(model.WorkspaceKindManagedSlot, model.ExecutorLocal, model.ItemBranchName("new-conversation", importTestID)))
	m.State = model.StateWorking
	occupiedID, _ := model.NewID()
	occupied := model.NewManifest(occupiedID, "fix-authentication-tests", "Occupied name", nil, false, time.Now(), source, model.NewCheckout(model.WorkspaceKindManagedSlot, model.ExecutorLocal, model.ItemBranchName("fix-authentication-tests", occupiedID)))
	for _, item := range []model.Manifest{m, occupied} {
		if err := native.CreateItem(context.Background(), item); err != nil {
			t.Fatal(err)
		}
	}
	db, err := OpenDatabase(root)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := seedManifests(context.Background(), db, root); err != nil {
		t.Fatal(err)
	}
	before, _ := db.CanonicalManifest(m.ID)
	cmd := ManifestCommand{ID: "finalize", ProtocolVersion: ProtocolVersion, Type: CommandNamingFinalize, ItemID: m.ID, ExpectedRevision: &before.Revision, Title: "Fix authentication tests", Actor: "user", CreatedAt: time.Now().UTC()}
	result, err := db.ExecuteManifestCommand(cmd)
	if err != nil {
		t.Fatal(err)
	}
	after := result.Manifest
	if after.ID != m.ID || !reflect.DeepEqual(after.Repository, m.Repository) || after.State != m.State || after.Checkout.Claim != nil || after.PrimaryConversation != nil {
		t.Fatalf("finalization changed identity/source/resources: %+v", after)
	}
	if after.Title != cmd.Title || !strings.HasPrefix(after.Slug, "fix-authentication-tests-") || after.Slug == occupied.Slug || after.Checkout.Specification.Branch != model.ItemBranchName(after.Slug, m.ID) {
		t.Fatalf("names not finalized together: %+v", after)
	}
	if len(result.Events) != 1 || result.Events[0].Type != "work_item.naming_finalized" {
		t.Fatalf("events=%+v", result.Events)
	}
	repeat, err := db.ExecuteManifestCommand(cmd)
	if err != nil || !repeat.Duplicate || repeat.Revision != result.Revision {
		t.Fatalf("replay=%+v err=%v", repeat, err)
	}
	stale := cmd
	stale.ID = "stale"
	stale.Title = "Do not overwrite"
	if _, err := db.ExecuteManifestCommand(stale); err == nil {
		t.Fatal("stale finalization applied")
	}
}
func TestNamingFinalizationCannotChangeMaterializedOrExplicitCheckout(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*model.Manifest)
	}{
		{"claimed", func(m *model.Manifest) { m.Checkout.Claim = &model.WorkspaceClaim{} }},
		{"previously materialized", func(m *model.Manifest) { m.Checkout.ClaimGeneration = 1 }},
		{"conversation exists", func(m *model.Manifest) { m.PrimaryConversation = &model.Conversation{ID: "existing"} }},
		{"explicit branch", func(m *model.Manifest) { m.Checkout.Specification.Branch = "chosen-by-user" }},
		{"repository home", func(m *model.Manifest) { m.Checkout.Specification.Kind = model.LegacyWorkspaceKindRepositoryHome }},
		{"waiting", func(m *model.Manifest) { m.State = model.StateWaiting }},
		{"archived", func(m *model.Manifest) { m.State = model.StateArchived }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := model.NewManifest(importTestID, "new-conversation", "New conversation", nil, false, time.Now(), model.Repository{}, model.NewCheckout(model.WorkspaceKindManagedSlot, model.ExecutorLocal, model.ItemBranchName("new-conversation", importTestID)))
			tc.change(&m)
			before := m
			if _, _, err := applyManifestCommand(ManifestCommand{Type: CommandNamingFinalize, Title: "Generated title", CreatedAt: time.Now()}, &m, []model.Manifest{m}); err == nil {
				t.Fatal("unsafe finalization accepted")
			}
			if !reflect.DeepEqual(m, before) {
				t.Fatal("rejected finalization changed names")
			}
		})
	}
}
