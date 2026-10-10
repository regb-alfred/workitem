package agentdriver

import (
	"context"
	"fmt"

	driverprotocol "github.com/regb/workitem/agentdriver/protocol"
)

func (c Client) History(ctx context.Context, path, leaf string) (driverprotocol.History, error) {
	res, err := c.request(ctx, driverprotocol.OperationConversationHistory, "", nil, driverprotocol.ConversationRequest{Path: path, LeafID: leaf})
	if err != nil {
		return driverprotocol.History{}, err
	}
	if res.History == nil {
		return driverprotocol.History{}, fmt.Errorf("driver does not provide readable history")
	}
	return *res.History, nil
}
