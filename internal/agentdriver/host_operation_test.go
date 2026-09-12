package agentdriver

import (
	"context"
	"os"
	"testing"

	driverprotocol "github.com/regb/workitem/agentdriver/protocol"
)

func TestRunOperationUsesDriverProcess(t *testing.T) {
	request := driverprotocol.Request{ProtocolVersion: driverprotocol.Version, Driver: "test", Operation: driverprotocol.OperationConversationCreate, Conversation: &driverprotocol.ConversationRequest{ID: "conversation-1"}}
	response, err := RunOperation(context.Background(), os.Args[0], []string{"-test.run=TestOperationHelperProcess"}, request, append(os.Environ(), "GO_WANT_WI_DRIVER_HELPER=1"))
	if err != nil {
		t.Fatal(err)
	}
	if response.Conversation == nil || response.Conversation.Conversation.Locator != "sessions/test/conversation-1.jsonl" {
		t.Fatalf("response=%+v", response)
	}
}

func TestOperationHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_WI_DRIVER_HELPER") != "1" {
		return
	}
	request, reply, err := driverprotocol.ConnectRequest(context.Background(), os.Getenv(driverprotocol.BootstrapSocketEnv), driverprotocol.Hello{ProtocolVersion: driverprotocol.Version, Driver: "test"})
	if err != nil {
		os.Exit(2)
	}
	conversation := driverprotocol.ConversationResponse{Conversation: driverprotocol.Conversation{ID: request.Conversation.ID, Locator: "sessions/test/" + request.Conversation.ID + ".jsonl", Format: "test-jsonl"}}
	if err := reply(driverprotocol.Response{ProtocolVersion: driverprotocol.Version, Conversation: &conversation}); err != nil {
		os.Exit(3)
	}
	os.Exit(0)
}
