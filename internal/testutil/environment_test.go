package testutil

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/regb/workitem/internal/xdg"
)

func TestIsolatedEnvironment(t *testing.T) {
	root := t.TempDir()
	base := []string{"HOME=/operator", "XDG_DATA_HOME=/operator/data", "XDG_RUNTIME_DIR=/operator/run", "WI_ID=live-item", "WI_DAEMON_SOCKET=/operator/daemon.sock", "WI_AGENT_DAEMON_SOCKET=/operator/agent.sock", "WI_AGENT_CONTROL_SOCKET=/operator/control.sock", "PI_CODING_AGENT_DIR=/operator/pi", "PI_SESSION_FILE=/operator/session", "TMUX=/operator/tmux", "TMUX_PANE=%1", "DIRENV_DIFF=host", "GIT_DIR=/operator/repo.git", "GOPATH=/operator/go", "GOBIN=/operator/bin", "PATH=/tools", "GOCACHE=/compiler-cache", "GOMODCACHE=/module-cache"}
	env := map[string]string{}
	for _, entry := range IsolatedEnvironment(base, root) {
		key, value, _ := strings.Cut(entry, "=")
		if _, exists := env[key]; exists {
			t.Fatalf("duplicate key %s", key)
		}
		env[key] = value
	}
	for _, key := range []string{"HOME", "XDG_DATA_HOME", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "XDG_STATE_HOME", "XDG_RUNTIME_DIR", "PI_CODING_AGENT_DIR", "GOPATH", "GOBIN"} {
		if !strings.HasPrefix(env[key], root+string(filepath.Separator)) {
			t.Fatalf("%s escaped test root: %s", key, env[key])
		}
	}
	for _, key := range []string{"WI_ID", "WI_DAEMON_SOCKET", "WI_AGENT_DAEMON_SOCKET", "WI_AGENT_CONTROL_SOCKET", "PI_SESSION_FILE", "TMUX", "TMUX_PANE", "DIRENV_DIFF", "GIT_DIR"} {
		if _, exists := env[key]; exists {
			t.Fatalf("inherited %s", key)
		}
	}
	if env["PATH"] != "/tools" || env["GOCACHE"] != "/compiler-cache" || env["GOMODCACHE"] != "/module-cache" {
		t.Fatalf("lost build tools or caches: %v", env)
	}
}

func TestIsolatedTestProcessDoesNotReachCallerSocket(t *testing.T) {
	root := ShortTempDir(t)
	socket := filepath.Join(root, "live.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	connections := make(chan bool, 1)
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			conn.Close()
		}
		connections <- err == nil
	}()
	env := []string{}
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "WI_") || strings.HasPrefix(key, "XDG_") || strings.HasPrefix(key, "PI_") || key == "HOME" {
			continue
		}
		env = append(env, entry)
	}
	env = append(env, "HOME="+root, "XDG_DATA_HOME="+root, "XDG_RUNTIME_DIR="+root, "WI_DAEMON_SOCKET="+socket, "WI_AGENT_DAEMON_SOCKET="+socket, "WI_ID=live", "PI_CODING_AGENT_DIR="+root, "GO_WANT_WI_ISOLATION_HELPER=1", "ISOLATION_REPORT="+filepath.Join(root, "report"))
	child := exec.Command(os.Args[0], "-test.run=^TestIsolatedEnvironmentHelper$")
	child.Env = env
	if output, err := child.CombinedOutput(); err != nil {
		t.Fatalf("helper: %v\n%s", err, output)
	}
	listener.Close()
	if <-connections {
		t.Fatal("test process connected to caller socket")
	}
	isolated, err := os.ReadFile(filepath.Join(root, "report"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(string(isolated)); !os.IsNotExist(err) {
		t.Fatalf("isolated root was not cleaned up: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "wi")); !os.IsNotExist(err) {
		t.Fatalf("test touched caller data root: %v", err)
	}
}

func TestIsolatedEnvironmentHelper(t *testing.T) {
	if os.Getenv("GO_WANT_WI_ISOLATION_HELPER") != "1" {
		return
	}
	code := RunIsolatedTests(func() int {
		root := os.Getenv(testProcessRoot)
		paths, err := xdg.FromEnv()
		if err != nil || !strings.HasPrefix(paths.DataRoot, root+"/") || !strings.HasPrefix(paths.RuntimeDir, root+"/") {
			t.Fatalf("paths=%+v error=%v", paths, err)
		}
		for _, key := range []string{"WI_DAEMON_SOCKET", "WI_AGENT_DAEMON_SOCKET", "WI_ID"} {
			if os.Getenv(key) != "" {
				t.Fatalf("inherited %s", key)
			}
		}
		if err := os.MkdirAll(paths.DataRoot, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(os.Getenv("ISOLATION_REPORT"), []byte(root), 0600); err != nil {
			t.Fatal(err)
		}
		return 0
	})
	if code != 0 {
		t.Fatalf("isolated tests failed: %d", code)
	}
}
