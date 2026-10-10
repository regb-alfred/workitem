package pi

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	driverprotocol "github.com/regb/workitem/agentdriver/protocol"
)

// ReadHistory selects one branch of the native entry tree. It does not change
// the file or open an AgentSession. The live TUI can supply its in-memory leaf.
func ReadHistory(ctx context.Context, path, leaf string) (driverprotocol.History, error) {
	result := driverprotocol.History{Messages: []driverprotocol.HistoryMessage{}}
	if leaf == "-" {
		return result, nil
	}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return result, err
	}
	if info.Size() > 64<<20 {
		return result, fmt.Errorf("history exceeds 64 MiB read limit")
	}
	type entry struct {
		Type      string         `json:"type"`
		ID        string         `json:"id"`
		Parent    string         `json:"parentId"`
		Timestamp string         `json:"timestamp"`
		Message   map[string]any `json:"message"`
		Summary   string         `json:"summary"`
	}
	entries := map[string]entry{}
	last := ""
	reader := bufio.NewReader(io.LimitReader(f, info.Size()))
	for {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		line, readErr := reader.ReadBytes('\n')
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return result, readErr
		}
		var e entry
		if json.Unmarshal(line, &e) != nil {
			return result, fmt.Errorf("malformed native conversation record")
		}
		if e.Type == "session" {
			continue
		}
		if e.ID == "" {
			return result, fmt.Errorf("legacy history without tree IDs is not supported")
		}
		entries[e.ID] = e
		last = e.ID
	}
	if leaf == "" {
		leaf = last
	}
	result.LeafID = leaf
	branch := []entry{}
	seen := map[string]bool{}
	for id := leaf; id != ""; {
		if seen[id] {
			return result, fmt.Errorf("conversation branch contains cycle")
		}
		seen[id] = true
		e, ok := entries[id]
		if !ok {
			return result, fmt.Errorf("conversation leaf/parent is not persisted yet")
		}
		branch = append(branch, e)
		id = e.Parent
	}
	total := 0
	for i := 0; i < len(branch); i++ {
		e := branch[i]
		role, _ := e.Message["role"].(string)
		text := ""
		switch e.Type {
		case "message":
			if role == "system" {
				continue
			}
			text = historyText(e.Message["content"])
			if role == "toolResult" {
				name, _ := e.Message["toolName"].(string)
				text = name + "\n" + text
			}
		case "compaction", "branch_summary":
			role = "summary"
			text = e.Summary
		default:
			continue
		}
		if len(text) > 16<<10 {
			end := 16 << 10
			for end > 0 && !utf8.RuneStart(text[end]) {
				end--
			}
			text = text[:end] + "\n[message truncated]"
			result.Truncated = true
		}
		if len(result.Messages) >= 200 || total+len(text) > 1<<20 {
			result.Truncated = true
			break
		}
		total += len(text)
		result.Messages = append(result.Messages, driverprotocol.HistoryMessage{ID: e.ID, Role: role, Text: text, Timestamp: e.Timestamp})
	}
	for i, j := 0, len(result.Messages)-1; i < j; i, j = i+1, j-1 {
		result.Messages[i], result.Messages[j] = result.Messages[j], result.Messages[i]
	}
	return result, nil
}
func historyText(content any) string {
	if s, ok := content.(string); ok {
		return s
	}
	blocks, _ := content.([]any)
	parts := []string{}
	for _, raw := range blocks {
		b, _ := raw.(map[string]any)
		typ, _ := b["type"].(string)
		switch typ {
		case "text":
			s, _ := b["text"].(string)
			parts = append(parts, s)
		case "toolCall":
			name, _ := b["name"].(string)
			parts = append(parts, "[tool call: "+name+"]")
		case "thinking":
			parts = append(parts, "[thinking omitted]")
		default:
			parts = append(parts, "["+typ+" content omitted]")
		}
	}
	return strings.Join(parts, "\n")
}
