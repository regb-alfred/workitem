package git

import (
	"context"
	"fmt"
)

// ReadBlob reads declarative project content without checking out the target ref.
func (c Client) ReadBlob(ctx context.Context, root, revision, path string) (string, error) {
	value, err := c.output(ctx, root, "show", revision+":"+path)
	if err != nil {
		return "", err
	}
	if len(value) > 1<<20 {
		return "", fmt.Errorf("project content exceeds 1 MiB")
	}
	return value, nil
}
