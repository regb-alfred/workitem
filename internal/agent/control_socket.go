package agent

import (
	"context"
	"errors"
	"fmt"

	driverprotocol "github.com/regb/workitem/agentdriver/protocol"
)

var ErrControlSocketUnavailable = errors.New("agent control socket unavailable")

type ControlSocketResponse = driverprotocol.ControlResponse
type ControlSocketRequest = driverprotocol.ControlRequest
type ControlSocketServer = driverprotocol.ControlServer

func ListenControlSocket(path string) (*ControlSocketServer, error) {
	return driverprotocol.ListenControl(path)
}

func SubmitControlSocket(ctx context.Context, path string, command ControlCommand) error {
	if err := driverprotocol.SubmitControl(ctx, path, command); err != nil {
		var rejected *driverprotocol.ControlRejectedError
		if errors.As(err, &rejected) {
			return rejected
		}
		return fmt.Errorf("%w: %v", ErrControlSocketUnavailable, err)
	}
	return nil
}
