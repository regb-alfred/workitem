package pi

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	driverprotocol "github.com/regb/workitem/agentdriver/protocol"
	"github.com/regb/workitem/internal/agent"
)

func TestHistorySelectsBranchAndIgnoresIncompleteAppend(t *testing.T) {
	path := filepath.Join(t.TempDir(), "native.jsonl")
	data := `{"type":"session","version":3,"id":"session"}
{"type":"message","id":"root","parentId":null,"message":{"role":"user","content":"hello"}}
{"type":"message","id":"abandoned","parentId":"root","message":{"role":"assistant","content":[{"type":"text","text":"old answer"}]}}
{"type":"message","id":"active","parentId":"root","message":{"role":"assistant","content":[{"type":"thinking","thinking":"private"},{"type":"text","text":"new answer"},{"type":"toolCall","name":"bash"}]}}
{"type":"message","id":"unfinished"`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := ReadHistory(context.Background(), path, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Messages) != 2 || result.LeafID != "active" || strings.Contains(result.Messages[1].Text, "old answer") || strings.Contains(result.Messages[1].Text, "private") {
		t.Fatalf("branch %+v", result)
	}
	result, err = ReadHistory(context.Background(), path, "abandoned")
	if err != nil || len(result.Messages) != 2 || result.Messages[1].Text != "old answer" {
		t.Fatalf("live TUI leaf %+v %v", result, err)
	}
	if _, err := ReadHistory(context.Background(), path, "missing"); err == nil {
		t.Fatal("missing leaf accepted")
	}
}
func TestHistoryRejectsMalformedAndCyclicInput(t *testing.T) {
	for _, data := range []string{"not json\n", `{"type":"message","id":"cycle","parentId":"cycle"}` + "\n", `{"type":"message","message":{"role":"user","content":"legacy"}}` + "\n"} {
		path := filepath.Join(t.TempDir(), "native.jsonl")
		_ = os.WriteFile(path, []byte(data), 0600)
		if _, err := ReadHistory(context.Background(), path, ""); err == nil {
			t.Fatal("malformed history accepted")
		}
	}
}
func TestHistoryLimitsAndPublicOperation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "native.jsonl")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	encoder := json.NewEncoder(f)
	parent := ""
	for i := 0; i < 250; i++ {
		id := strings.Repeat("x", i+1)
		_ = encoder.Encode(map[string]any{"type": "message", "id": id, "parentId": parent, "message": map[string]any{"role": "user", "content": "hello"}})
		parent = id
	}
	_ = f.Close()
	response, err := handleConversationOperation(context.Background(), driverprotocol.Request{Operation: driverprotocol.OperationConversationHistory, Conversation: &driverprotocol.ConversationRequest{Path: path}})
	if err != nil {
		t.Fatal(err)
	}
	if response.History == nil || !response.History.Truncated || len(response.History.Messages) != 200 {
		t.Fatalf("history limit %+v", response.History)
	}
}
func TestNativeRPCIdleGuardRefusesBeforeWritingOrStopping(t *testing.T) {
	transport := &nativeRPCTransport{spec: ExecSpec{RuntimeID: "owner", WorkItemID: "item"}, busy: true, handled: map[string]bool{}}
	for _, typ := range []string{agent.CommandPrompt, agent.CommandShutdown} {
		err := transport.sendCommand(agent.ControlCommand{ProtocolVersion: 1, ID: typ, RuntimeID: "owner", WorkItemID: "item", Type: typ, RequireIdle: true})
		if err == nil {
			t.Fatal("busy control accepted")
		}
		if transport.stopping || transport.handled[typ] {
			t.Fatal("refused command changed runtime")
		}
	}
}
