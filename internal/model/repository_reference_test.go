package model_test

import (
	"encoding/json"
	"github.com/regb/workitem/internal/model"
	"strings"
	"testing"
)

func TestRepositoryReferenceDoesNotPersistOperationalLocation(t *testing.T) {
	r := model.Repository{ID: "01M17E1HRA465ERBR8H5Z4F68E", Name: "wi", RootAtCreation: "/external/source", CurrentRoot: "/other", GitCommonDir: "/bare.git", RemoteURL: "git@example:repo", CreatedFromCommit: strings.Repeat("a", 40)}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(b, &fields); err != nil {
		t.Fatal(err)
	}
	if len(fields) != 2 || fields["id"] != r.ID {
		t.Fatalf("stored fields: %s", b)
	}
	if model.RepositoryKey(r) != r.ID {
		t.Fatal("repository key depends on location")
	}
	var decoded model.Repository
	if err := json.Unmarshal([]byte(`{"id":"01M17E1HRA465ERBR8H5Z4F68E","root_at_creation":"/old"}`), &decoded); err == nil {
		t.Fatal("accepted copied location in a registered reference")
	}
}
