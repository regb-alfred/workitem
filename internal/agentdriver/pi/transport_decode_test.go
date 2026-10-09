package pi

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/regb/workitem/internal/agentdriver"
	"github.com/regb/workitem/internal/model"
)

func TestDecodeTransportPreservesMalformedNativeLines(t *testing.T) {
	client := agentdriver.Client{Driver: "pi", Path: os.Args[0], Args: []string{"-test.run=^TestDecodeTransportHelper$"}, Env: append(os.Environ(), "GO_WANT_WI_PI_DECODE_HELPER=1")}
	lines := [][]byte{
		[]byte(`{"type":"message","message":{"role":"user","content":"question"}}`),
		[]byte(`{"type":"message",damaged`),
		{0xff, 0x00, '\n'},
		[]byte(`{"type":"message","message":{"role":"assistant","content":"answer","stopReason":"stop"}}`),
	}
	facts, decoded, err := client.DecodeConversation(context.Background(), model.Conversation{Driver: "pi", ID: "conversation", Format: "pi-jsonl"}, []uint64{11, 12, 13, 14}, lines)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, []bool{true, false, false, true}) || len(facts) != 2 || facts[0].Line != 11 || facts[1].Line != 14 || !facts[1].Terminal {
		t.Fatalf("facts=%+v decoded=%v", facts, decoded)
	}
}

func TestDecodeTransportHelper(t *testing.T) {
	if os.Getenv("GO_WANT_WI_PI_DECODE_HELPER") != "1" {
		return
	}
	if err := Run(context.Background()); err != nil {
		os.Exit(2)
	}
	os.Exit(0)
}
