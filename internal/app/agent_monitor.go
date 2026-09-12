package app

import (
	"context"
	"time"

	agentcore "github.com/regb/workitem/internal/app/core/primaryagent"
	"github.com/regb/workitem/internal/model"
)

type AgentStatusOptions struct {
	ResolveOptions
	StaleAfter time.Duration
}
type AgentStatusResult = agentcore.AgentStatusResult
type AgentProcessStatus = agentcore.AgentProcessStatus
type ConversationStatus = agentcore.ConversationStatus
type ConversationEventSummary = agentcore.ConversationEventSummary
type WorktreeStatus = agentcore.WorktreeStatus

func (a *App) AgentStatus(ctx context.Context, opts AgentStatusOptions) (AgentStatusResult, error) {
	return a.primaryAgentService().AgentStatus(ctx, agentcore.AgentStatusOptions{ResolveOptions: opts.ResolveOptions, StaleAfter: opts.StaleAfter})
}
func (a *App) recordPrimaryTerminalRuntime(ctx context.Context, m model.Manifest, window string) []string {
	return a.primaryAgentService().RecordPrimaryTerminalRuntime(ctx, m, window)
}
func (a *App) inspectConversation(ctx context.Context, m model.Manifest, now time.Time) (ConversationStatus, []string) {
	if a.ConversationProjectionObserver != nil {
		return a.ConversationProjectionObserver(ctx, m, now)
	}
	return a.observeConversation(ctx, m, now)
}
