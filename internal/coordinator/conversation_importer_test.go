package coordinator

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/regb/workitem/internal/model"
)

func TestConversationImporterIndexesIncrementallyWithoutConversationContent(t *testing.T) {
	root, itemID, sessionID := t.TempDir(), importTestID, "01KZYHHK91GWS6FZFQZQTT1KJ2"
	relative := filepath.Join("sessions", "pi", sessionID+".jsonl")
	path := filepath.Join(root, "items", itemID, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	at := time.Now().UTC().Truncate(time.Microsecond)
	values := []map[string]any{
		{"type": "message", "timestamp": at.Format(time.RFC3339Nano), "message": map[string]any{"role": "user", "content": []map[string]any{{"type": "text", "text": "PRIVATE_USER_PROMPT"}}}},
		{"type": "message", "timestamp": at.Add(time.Second).Format(time.RFC3339Nano), "message": map[string]any{"role": "assistant", "content": []map[string]any{{"type": "toolCall", "name": "bash", "arguments": map[string]any{"secret": "PRIVATE_TOOL_ARGUMENT"}}}}},
		{"type": "toolResult", "timestamp": at.Add(2 * time.Second).Format(time.RFC3339Nano), "message": map[string]any{"role": "toolResult", "content": []map[string]any{{"type": "text", "text": "PRIVATE_TOOL_RESULT"}}}},
		{"type": "message", "timestamp": at.Add(3 * time.Second).Format(time.RFC3339Nano), "message": map[string]any{"role": "assistant", "stopReason": "stop", "content": []map[string]any{{"type": "text", "text": "PRIVATE_ASSISTANT_RESPONSE"}}}},
	}
	var contents bytes.Buffer
	for _, value := range values {
		encoded, _ := json.Marshal(value)
		contents.Write(encoded)
		contents.WriteByte('\n')
	}
	if err := os.WriteFile(path, contents.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	db, err := OpenDatabase(root)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	manifest := model.NewManifest(itemID, "pi-index", "Pi index", nil, false, at, model.Repository{}, model.Checkout{})
	manifest.PrimaryConversation = &model.Conversation{Driver: "pi", ID: sessionID, Locator: relative}
	encodedManifest, _ := json.Marshal(manifest)
	digest := sha256.Sum256(encodedManifest)
	if _, err := db.SyncImportedManifests(map[string]ImportedManifest{itemID: {Manifest: manifest, Digest: hex.EncodeToString(digest[:])}}, true); err != nil {
		t.Fatal(err)
	}
	first, err := ImportConversations(context.Background(), db, root, testConversationDecoders)
	if err != nil || first.FilesImported != 1 || first.EntriesScanned != 4 || first.MalformedLines != 0 {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	var index ConversationIndex
	if ok, err := db.ReadProjection(ConversationProjection, itemID, &index); err != nil || !ok || index.InferredTurnState != "idle" || index.LastUserPrompt == nil || index.LastTerminalAssistant == nil || index.LastToolActivity == nil {
		t.Fatalf("index=%+v ok=%v err=%v", index, ok, err)
	}
	var activity AttentionActivity
	if ok, err := db.ReadProjection(AttentionActivityProjection, itemID, &activity); err != nil || !ok || activity.LastRequestedAt == nil || activity.LastCompletedAt == nil {
		t.Fatalf("activity=%+v ok=%v err=%v", activity, ok, err)
	}
	databaseBytes, err := os.ReadFile(DatabasePath(root))
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"PRIVATE_USER_PROMPT", "PRIVATE_TOOL_ARGUMENT", "PRIVATE_TOOL_RESULT", "PRIVATE_ASSISTANT_RESPONSE"} {
		if bytes.Contains(databaseBytes, []byte(secret)) {
			t.Fatalf("database contains Pi content %q", secret)
		}
	}
	second, err := ImportConversations(context.Background(), db, root, testConversationDecoders)
	if err != nil || second.EntriesScanned != 0 || second.BytesRead != 0 {
		t.Fatalf("second=%+v err=%v", second, err)
	}
}

var testConversationDecoders = ConversationDecoders{"pi": func(_ context.Context, conversation model.Conversation, records []ConversationRecord) ([]ConversationEventFact, []bool, error) {
	facts := make([]ConversationEventFact, 0, len(records))
	decoded := make([]bool, len(records))
	for i, record := range records {
		fact, ok := testConversationLineDecoder(conversation, record.Line, record.Raw)
		decoded[i] = ok
		if ok {
			facts = append(facts, fact)
		}
	}
	return facts, decoded, nil
}}

func testConversationLineDecoder(_ model.Conversation, line uint64, raw []byte) (ConversationEventFact, bool) {
	var value struct {
		Type      string `json:"type"`
		Timestamp string `json:"timestamp"`
		Message   struct {
			Role       string `json:"role"`
			StopReason string `json:"stopReason"`
			Content    []struct {
				Type string `json:"type"`
			} `json:"content"`
		} `json:"message"`
	}
	if json.Unmarshal(raw, &value) != nil {
		return ConversationEventFact{}, false
	}
	fact := ConversationEventFact{Line: line, Type: value.Type, Role: value.Message.Role, StopReason: value.Message.StopReason}
	fact.Timestamp, _ = time.Parse(time.RFC3339Nano, value.Timestamp)
	for _, block := range value.Message.Content {
		fact.ContentTypes = append(fact.ContentTypes, block.Type)
	}
	if fact.Role == "assistant" && fact.StopReason == "stop" {
		fact.Terminal = true
	}
	return fact, true
}
