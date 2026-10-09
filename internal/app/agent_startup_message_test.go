package app_test

import (
	"github.com/regb/workitem/internal/app"
	"github.com/regb/workitem/internal/config"
	"testing"
)

func TestAgentStartupMessagePrecedence(t *testing.T) {
	a := &app.App{}
	if got := a.AgentStartupMessage(""); got != config.DefaultStartupMessage {
		t.Fatalf("default=%q", got)
	}
	a.AgentConfig.StartupMessage = "  Read wi show and check the acceptance criteria.  "
	if got := a.AgentStartupMessage(""); got != "Read wi show and check the acceptance criteria." {
		t.Fatalf("configured=%q", got)
	}
	if got := a.AgentStartupMessage("  Start with the regression test.  "); got != "Start with the regression test." {
		t.Fatalf("override=%q", got)
	}
}
