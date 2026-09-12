package model

import (
	"encoding/json"
	"testing"
)

func TestManifestUsesGenericConversationFields(t *testing.T) {
	manifest := Manifest{ID: "01KZYHHK91GWS6FZFQZQTT1KJ2", Agent: "default", PrimaryConversation: &Conversation{Driver: "pi", ID: "conversation-1", Locator: "sessions/pi/conversation-1.jsonl", Format: "pi-jsonl"}}
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if fields["primary_conversation"] == nil {
		t.Fatalf("manifest fields=%v", fields)
	}
	var conversation map[string]json.RawMessage
	if err := json.Unmarshal(fields["primary_conversation"], &conversation); err != nil {
		t.Fatal(err)
	}
	if conversation["locator"] == nil || conversation["path"] != nil {
		t.Fatalf("conversation fields=%v", conversation)
	}
}
