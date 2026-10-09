package app

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/regb/workitem/internal/model"
)

func TestAgentRuntimeEnvironmentKeepsWiOnPath(t *testing.T) {
	for _, tt := range []struct{ name, self, path, project, want string }{
		{"stale server", "/home/user/go/bin/wi", "/clean/bin", "", "/home/user/go/bin:/clean/bin"},
		{"already present", "/home/user/go/bin/wi", "/clean/bin:/home/user/go/bin", "", "/clean/bin:/home/user/go/bin"},
		{"direnv replacement", "/home/user/go/bin/wi", "/clean/bin", "/project/bin", "/home/user/go/bin:/project/bin"},
		{"empty path", "/home/user/go/bin/wi", "", "", "/home/user/go/bin"},
		{"unset executable", "", "/clean/bin", "", "/clean/bin"},
		{"relative executable", "wi", "/clean/bin", "", "/clean/bin"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PATH", "/other-item/bin")
			path := t.TempDir()
			d := &runtimeDirenv{status: model.DirenvStatus{Found: tt.project != "", Allowed: true, RCPath: filepath.Join(path, ".envrc")}, env: map[string]string{"PATH": tt.project}}
			a := &App{SelfPath: tt.self, Direnv: d, Tmux: fakeBaseTmux{global: map[string]string{"PATH": tt.path, "HOME": "/home/user"}}}
			m := model.Manifest{Checkout: model.Checkout{ClaimGeneration: 1, Claim: &model.WorkspaceClaim{ID: "test-claim", Generation: 1, Executor: model.ExecutorLocal, Path: path}}}
			env, _, _, err := a.agentRuntimeEnvironment(context.Background(), m)
			if err != nil || env["PATH"] != tt.want {
				t.Fatalf("path=%q want=%q error=%v", env["PATH"], tt.want, err)
			}
			if slices.Contains(filepath.SplitList(env["PATH"]), "/other-item/bin") {
				t.Fatalf("imported caller PATH %s", env["PATH"])
			}
			if tt.self != "" && filepath.IsAbs(tt.self) && strings.Count(env["PATH"], filepath.Dir(tt.self)) != 1 {
				t.Fatalf("duplicated executable directory %q", env["PATH"])
			}
		})
	}
}
