package agentdriver

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/regb/workitem/agentdriver/protocol"
)

const driverBootstrapTimeout = 10 * time.Second

// Run starts one trusted driver process and supplies its launch request over a
// private bootstrap socket. Standard input and output remain attached so TUI
// drivers can hand them to their backend.
func Run(ctx context.Context, path string, args []string, request driverprotocol.LaunchRequest, env []string, onReady func(driverprotocol.Ready) error) error {
	if path == "" {
		return fmt.Errorf("agent driver executable is required")
	}
	bootstrapPath := filepath.Join(filepath.Dir(request.Runtime.ControlSocketPath), "driver.sock")
	if request.Runtime.ControlSocketPath == "" {
		return fmt.Errorf("agent driver bootstrap requires a runtime socket path")
	}
	server, err := driverprotocol.Listen(bootstrapPath)
	if err != nil {
		return fmt.Errorf("listen for agent driver: %w", err)
	}
	defer server.Close()

	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Dir = request.Runtime.CWD
	cmd.Env = withDriverBootstrapSocket(env, bootstrapPath)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start agent driver %s: %w", filepath.Base(path), err)
	}
	readyCtx, cancel := context.WithTimeout(ctx, driverBootstrapTimeout)
	ready, handshakeErr := server.ServeOnce(readyCtx, request)
	cancel()
	if handshakeErr != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return handshakeErr
	}
	switch request.Runtime.Mode {
	case driverprotocol.ModeTUI:
		if !ready.Capabilities.NativeTUI {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return fmt.Errorf("agent driver %q does not support TUI mode", request.Driver)
		}
	case driverprotocol.ModeHeadless:
		if !ready.Capabilities.Headless {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return fmt.Errorf("agent driver %q does not support headless mode", request.Driver)
		}
	default:
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return fmt.Errorf("unsupported runtime mode %q", request.Runtime.Mode)
	}
	if onReady != nil {
		if err := onReady(ready); err != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return err
		}
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("agent driver %s: %w", filepath.Base(path), err)
	}
	return nil
}

func RunOperation(ctx context.Context, path string, args []string, request driverprotocol.Request, env []string) (driverprotocol.Response, error) {
	if path == "" {
		return driverprotocol.Response{}, fmt.Errorf("agent driver executable is required")
	}
	dir, err := os.MkdirTemp("", "wi-driver-")
	if err != nil {
		return driverprotocol.Response{}, err
	}
	defer os.RemoveAll(dir)
	bootstrapPath := filepath.Join(dir, "driver.sock")
	server, err := driverprotocol.Listen(bootstrapPath)
	if err != nil {
		return driverprotocol.Response{}, fmt.Errorf("listen for agent driver: %w", err)
	}
	defer server.Close()
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Env = withDriverBootstrapSocket(env, bootstrapPath)
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return driverprotocol.Response{}, fmt.Errorf("start agent driver %s: %w", filepath.Base(path), err)
	}
	response, operationErr := server.ServeRequestOnce(ctx, request)
	if operationErr != nil {
		_ = cmd.Process.Kill()
	}
	waitErr := cmd.Wait()
	if operationErr != nil {
		return response, operationErr
	}
	if waitErr != nil {
		return response, fmt.Errorf("agent driver %s: %w", filepath.Base(path), waitErr)
	}
	return response, nil
}

func withDriverBootstrapSocket(env []string, path string) []string {
	key := driverprotocol.BootstrapSocketEnv + "="
	result := make([]string, 0, len(env)+1)
	for _, value := range env {
		if len(value) >= len(key) && value[:len(key)] == key {
			continue
		}
		result = append(result, value)
	}
	return append(result, key+path)
}
