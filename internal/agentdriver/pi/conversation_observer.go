package pi

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"time"

	driverprotocol "github.com/regb/workitem/agentdriver/protocol"
	agentcore "github.com/regb/workitem/internal/app/core/primaryagent"
	"github.com/regb/workitem/internal/model"
)

func ObserveConversationPath(path string, now time.Time) (driverprotocol.ConversationObservation, []string) {
	status := driverprotocol.ConversationObservation{Source: "driver.pi"}
	info, err := os.Stat(path)
	if err != nil {
		status.UnavailableReason = "conversation file is unavailable: " + err.Error()
		return status, nil
	}
	status.Exists, status.SizeBytes, status.ModTime = true, info.Size(), info.ModTime()
	file, err := os.Open(path)
	if err != nil {
		status.UnavailableReason = "conversation file is unavailable: " + err.Error()
		return status, nil
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 16<<20)
	line := uint64(0)
	for scanner.Scan() {
		line++
		fact, ok := SummarizeConversationFact(line, scanner.Bytes())
		if !ok {
			status.MalformedLines++
			continue
		}
		status.EntriesScanned++
		status.LastEvent = &fact
		turnActivity := fact.Role == "user" || fact.Role == "assistant" || fact.Role == "toolResult" || fact.Type == "toolResult" || containsType(fact.ContentTypes, "toolCall", "toolResult", "bashExecution")
		if turnActivity {
			status.LastTurnActivity = &fact
		}
		switch fact.Role {
		case "user":
			status.LastUserPrompt = &fact
		case "assistant":
			status.LastAssistantMessage = &fact
			if fact.Terminal || fact.Failed {
				status.LastTerminalAssistant = &fact
			}
		case "toolResult":
			status.LastToolActivity = &fact
		}
		if fact.Role == "assistant" && containsType(fact.ContentTypes, "toolCall", "bashExecution") {
			status.LastToolActivity = &fact
		}
	}
	warnings := []string{}
	if err := scanner.Err(); err != nil {
		warnings = append(warnings, "could not fully scan conversation: "+err.Error())
	}
	if status.LastTurnActivity != nil && !status.LastTurnActivity.Timestamp.IsZero() {
		status.LastActivityAgeSeconds = int64(now.Sub(status.LastTurnActivity.Timestamp).Seconds())
	}
	status.InferredTurnState = inferTurnState(status)
	return status, warnings
}

// ObserveConversation is retained as an adapter for callers migrating to the
// process-backed driver client.
func ObserveConversation(itemDir func(string) string) agentcore.ConversationObserver {
	return func(_ context.Context, manifest model.Manifest, now time.Time) (agentcore.ConversationStatus, []string) {
		status := agentcore.ConversationStatus{Source: "driver.pi"}
		if manifest.PrimaryConversation == nil {
			status.UnavailableReason = "no primary conversation is recorded"
			return status, nil
		}
		conversation := *manifest.PrimaryConversation
		path := filepath.Join(itemDir(manifest.ID), filepath.FromSlash(conversation.NativeLocator()))
		observed, warnings := ObserveConversationPath(path, now)
		status = toCoreObservation(conversation, path, observed)
		return status, warnings
	}
}

func toCoreObservation(conversation model.Conversation, path string, observed driverprotocol.ConversationObservation) agentcore.ConversationStatus {
	result := agentcore.ConversationStatus{ID: conversation.ID, Path: path, PathRel: conversation.NativeLocator(), Source: observed.Source, Exists: observed.Exists, SizeBytes: observed.SizeBytes, ModTime: observed.ModTime, EntriesScanned: observed.EntriesScanned, MalformedLines: observed.MalformedLines, LastActivityAgeSeconds: observed.LastActivityAgeSeconds, InferredTurnState: observed.InferredTurnState, UnavailableReason: observed.UnavailableReason}
	convert := func(fact *driverprotocol.ConversationFact) *agentcore.ConversationEventSummary {
		if fact == nil {
			return nil
		}
		return &agentcore.ConversationEventSummary{Line: int(fact.Line), Type: fact.Type, Timestamp: fact.Timestamp, Role: fact.Role, StopReason: fact.StopReason, ContentTypes: fact.ContentTypes, Terminal: fact.Terminal, Failed: fact.Failed}
	}
	result.LastEvent = convert(observed.LastEvent)
	result.LastTurnActivity = convert(observed.LastTurnActivity)
	result.LastUserPrompt = convert(observed.LastUserPrompt)
	result.LastAssistantMessage = convert(observed.LastAssistantMessage)
	result.LastTerminalAssistant = convert(observed.LastTerminalAssistant)
	result.LastToolActivity = convert(observed.LastToolActivity)
	return result
}

func containsType(values []string, needles ...string) bool {
	for _, value := range values {
		for _, needle := range needles {
			if value == needle {
				return true
			}
		}
	}
	return false
}

func inferTurnState(status driverprotocol.ConversationObservation) string {
	if status.LastTurnActivity == nil {
		return "idle"
	}
	if terminal := status.LastTerminalAssistant; terminal != nil && (status.LastUserPrompt == nil || !status.LastUserPrompt.Timestamp.After(terminal.Timestamp)) && (status.LastToolActivity == nil || !status.LastToolActivity.Timestamp.After(terminal.Timestamp)) {
		if terminal.Failed {
			return "failed"
		}
		return "idle"
	}
	return "incomplete"
}
