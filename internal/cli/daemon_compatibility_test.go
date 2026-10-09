package cli

import (
	"context"
	"encoding/json"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/regb/workitem/internal/coordinator"
	"github.com/regb/workitem/internal/testutil"
	"github.com/regb/workitem/internal/xdg"
)

func TestEnsureDaemonRefusesIncompatibleDatabaseWithoutShutdown(t *testing.T) {
	for _, schema := range []uint64{0, coordinator.SchemaVersion - 1, coordinator.SchemaVersion + 1} {
		root := testutil.ShortTempDir(t)
		socket := filepath.Join(root, "daemon.sock")
		listener, err := net.Listen("unix", socket)
		if err != nil {
			t.Fatal(err)
		}
		methods := make(chan string, 16)
		done := make(chan struct{})
		go func() {
			defer close(done)
			for {
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				conn.SetDeadline(time.Now().Add(time.Second))
				var request coordinator.Request
				if json.NewDecoder(conn).Decode(&request) == nil {
					methods <- request.Method
					status := coordinator.Status{Database: coordinator.DatabaseStatus{SchemaVersion: schema}}
					payload, _ := json.Marshal(status)
					json.NewEncoder(conn).Encode(coordinator.Response{ProtocolVersion: coordinator.ProtocolVersion, BuildIdentity: "other-build", ID: request.ID, OK: true, Result: payload})
				}
				conn.Close()
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_, started, err := ensureDaemonRunning(ctx, xdg.Paths{DataStateRoot: filepath.Join(root, "state")}, socket)
		cancel()
		listener.Close()
		<-done
		close(methods)
		if err == nil || started || !strings.Contains(err.Error(), "cannot automatically replace daemon schema") {
			t.Fatalf("schema=%d started=%v error=%v", schema, started, err)
		}
		for method := range methods {
			if method != coordinator.MethodStatus {
				t.Fatalf("schema=%d sent %s", schema, method)
			}
		}
	}
}
