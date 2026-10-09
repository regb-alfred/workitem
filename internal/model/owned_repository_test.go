package model_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/regb/workitem/internal/model"
)

func TestOwnedRepositoryLocationIsDerivedFromIdentity(t *testing.T) {
	id := "01M17E1HRA465ERBR8H5Z4F68E"
	for _, root := range []string{"/first/data", "/moved/data"} {
		path, err := model.OwnedRepositoryGitDir(root, id)
		if err != nil || path != filepath.Join(root, "repositories", id+".git") {
			t.Fatalf("path=%s error=%v", path, err)
		}
	}
	if _, err := model.OwnedRepositoryGitDir("/data", "../escape"); err == nil {
		t.Fatal("accepted unsafe identity")
	}
	if _, err := model.OwnedRepositoryGitDir("relative/data", id); err == nil {
		t.Fatal("accepted relative data root")
	}
}

func TestOwnedRepositoryRejectsCredentialBearingSources(t *testing.T) {
	for _, source := range []string{"", "https://user:password@example.com/repo.git", "https://example.com/repo.git?token=secret", "ssh://user:password@example.com/repo.git", "path\nwith-control"} {
		if err := model.ValidateRepositorySource(source); err == nil {
			t.Fatalf("accepted source %q", source)
		}
	}
	for _, source := range []string{"/source/repo", "https://example.com/repo.git", "git@example.com:owner/repo.git", "ssh://git@example.com/repo.git"} {
		if err := model.ValidateRepositorySource(source); err != nil {
			t.Fatalf("rejected source %q: %v", source, err)
		}
	}
}

func TestOwnedRepositoryRequiresExplicitNameAndIntegrationBranch(t *testing.T) {
	record := model.OwnedRepository{ID: "01M17E1HRA465ERBR8H5Z4F68E", Name: "fleet", IntegrationBranch: "main", CreatedAt: time.Now()}
	if _, err := model.NormalizeOwnedRepository(record); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"", "../fleet", "Fleet", "two names", "/fleet"} {
		bad := record
		bad.Name = name
		if _, err := model.NormalizeOwnedRepository(bad); err == nil {
			t.Fatalf("accepted name %q", name)
		}
	}
	for _, branch := range []string{"", "../main", "main.lock", "-main", "main~1", "main@{1}", "foo//bar", ".hidden", "trailing.", "main\nother"} {
		bad := record
		bad.IntegrationBranch = branch
		if _, err := model.NormalizeOwnedRepository(bad); err == nil {
			t.Fatalf("accepted branch %q", branch)
		}
	}
}
