package app

import (
	"context"
	"testing"

	"github.com/regb/workitem/internal/agent"
	"github.com/regb/workitem/internal/model"
)

type forbiddenEnvironmentTmux struct {
	fakeBaseTmux
	t *testing.T
}

func (f forbiddenEnvironmentTmux) GlobalEnvironment(context.Context) (map[string]string, error) {
	f.t.Error("headless inspected tmux global environment")
	return nil, nil
}
func (f forbiddenEnvironmentTmux) SessionEnvironment(context.Context, string) (map[string]string, error) {
	f.t.Error("headless inspected tmux session environment")
	return nil, nil
}
func TestHeadlessEnvironmentPreservesCallerDataRootWithoutTmux(t *testing.T) {
	home := t.TempDir()
	data := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", data)
	t.Setenv("PI_SESSION_ID", "parent-conversation")
	t.Setenv("WI_ID", "parent-item")
	a := &App{Tmux: forbiddenEnvironmentTmux{t: t}}
	env, _, _, err := a.agentRuntimeEnvironment(context.Background(), model.Manifest{}, agent.ModeHeadless)
	if err != nil {
		t.Fatal(err)
	}
	if env["HOME"] != home || env["XDG_DATA_HOME"] != data || env["PI_SESSION_ID"] != "" || env["WI_ID"] != "" {
		t.Fatalf("unexpected headless root/identity HOME=%s XDG_DATA_HOME=%s PI_SESSION_ID=%s WI_ID=%s", env["HOME"], env["XDG_DATA_HOME"], env["PI_SESSION_ID"], env["WI_ID"])
	}
}
