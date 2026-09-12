package primaryagent

import (
	"strings"

	"github.com/regb/workitem/internal/model"
)

func applyPaneInfo(workspace *model.TerminalRuntime, pane model.TerminalPaneInfo) {
	workspace.TmuxWindow = firstNonEmpty(pane.WindowName, workspace.TmuxWindow)
	workspace.TmuxPaneID = pane.PaneID
	workspace.TmuxPaneIndex = pane.PaneIndex
	workspace.TmuxPanePID = pane.PanePID
	workspace.TmuxPaneCommand = pane.Command
	workspace.TmuxPanePath = pane.CurrentPath
}

func applyPaneStatus(status *AgentProcessStatus, pane model.TerminalPaneInfo) {
	status.TmuxSession = firstNonEmpty(pane.SessionName, status.TmuxSession)
	status.TmuxWindow = firstNonEmpty(pane.WindowName, status.TmuxWindow)
	status.TmuxPaneID = pane.PaneID
	status.TmuxPaneIndex = pane.PaneIndex
	status.TmuxPanePID = pane.PanePID
	status.TmuxPaneCommand = pane.Command
	status.TmuxPanePath = pane.CurrentPath
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func containsAny(values []string, needles ...string) bool {
	for _, value := range values {
		for _, needle := range needles {
			if value == needle {
				return true
			}
		}
	}
	return false
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func truncatePreview(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
