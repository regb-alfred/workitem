package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/regb/workitem/internal/coordinator"
	gitpkg "github.com/regb/workitem/internal/git"
	"github.com/regb/workitem/internal/model"
)

type baseRepositoryInspector interface {
	DetectRepository(context.Context, string, string) (model.RepositoryInfo, error)
}

func resolveRepositoryBase(ctx context.Context, revision, cwd string, current *model.Manifest, selected coordinator.RepositorySourceResult, git baseRepositoryInspector) (string, error) {
	if revision == "^" || revision == "" {
		return "", nil
	}
	relative, contextual := gitpkg.WorktreeRelativeRevision(revision)
	if !contextual {
		return revision, nil
	}
	if current == nil || current.Checkout.Claim == nil || current.Checkout.Claim.Path == "" {
		return "", fmt.Errorf("base %q requires a current work item with a materialized worktree; select a named branch, tag, commit, or --base '^' instead", revision)
	}
	if current.Repository.ID != selected.Repository.ID {
		return "", fmt.Errorf("base %q belongs to the current worktree's repository, not selected repository %s", revision, selected.Repository.Name)
	}
	root, err := filepath.EvalSymlinks(current.Checkout.Claim.Path)
	if err != nil {
		return "", fmt.Errorf("inspect current worktree: %w", err)
	}
	location, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		return "", fmt.Errorf("inspect current directory: %w", err)
	}
	distance, err := filepath.Rel(root, location)
	if err != nil || distance == ".." || strings.HasPrefix(distance, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("base %q requires running inside the current item's claimed worktree", revision)
	}
	info, err := git.DetectRepository(ctx, location, relative)
	if err != nil {
		return "", fmt.Errorf("resolve base %q in current worktree: %w", revision, err)
	}
	common, err := filepath.EvalSymlinks(info.Repository.GitCommonDir)
	if err != nil {
		return "", fmt.Errorf("inspect worktree Git store: %w", err)
	}
	store, err := filepath.EvalSymlinks(selected.GitDir)
	if err != nil {
		return "", fmt.Errorf("inspect selected Git store: %w", err)
	}
	if common != store {
		return "", fmt.Errorf("current directory is not a worktree of selected repository %s", selected.Repository.Name)
	}
	if info.Commit == "" {
		return "", fmt.Errorf("base %q did not resolve to a commit", revision)
	}
	return info.Commit, nil
}
