package app

import (
	"context"

	itemlock "github.com/regb/workitem/internal/lock"
	"github.com/regb/workitem/internal/model"
)

// lockItemOperation serializes host-side resource orchestration across processes.
// Acquire it before any store/native locks. Internal steps use unlocked helpers
// so composed use cases never recursively acquire the operation lock.
func (a *App) lockItemOperation(ctx context.Context, opts ResolveOptions) (model.Manifest, *itemlock.FileLock, error) {
	m, err := a.ResolveItem(ctx, opts)
	if err != nil {
		return model.Manifest{}, nil, err
	}
	operation, err := itemlock.AcquireOperation(ctx, a.Store.LockPath(m.ID))
	if err != nil {
		return model.Manifest{}, nil, err
	}
	m, err = a.Store.LoadManifest(m.ID)
	if err != nil {
		_ = operation.Release()
		return model.Manifest{}, nil, err
	}
	return m, operation, nil
}
