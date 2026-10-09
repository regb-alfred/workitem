package pi

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	protocol "github.com/regb/workitem/agentdriver/protocol"
)

// exportSkills creates private runtime resources, never global Pi configuration.
// The caller keeps them alive until the backend exits and then calls cleanup.
func exportSkills(ctx context.Context, skills []protocol.Skill) (paths []string, cleanup func(), err error) {
	if err := protocol.ValidateSkills(skills); err != nil {
		return nil, nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if len(skills) == 0 {
		return nil, func() {}, nil
	}
	root, err := os.MkdirTemp("", "wi-pi-skills-")
	if err != nil {
		return nil, nil, fmt.Errorf("create temporary Pi skills: %w", err)
	}
	remove := func() { _ = os.RemoveAll(root) }
	defer func() {
		if err != nil {
			remove()
		}
	}()
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, nil, err
	}
	root = absoluteRoot
	for _, skill := range skills {
		dir := filepath.Join(root, skill.Name)
		for _, file := range skill.Files {
			if err := ctx.Err(); err != nil {
				return nil, nil, err
			}
			path := filepath.Join(dir, filepath.FromSlash(file.Path))
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				return nil, nil, err
			}
			mode := os.FileMode(0600)
			if file.Executable {
				mode = 0700
			}
			if err := os.WriteFile(path, file.Data, mode); err != nil {
				return nil, nil, err
			}
		}
		paths = append(paths, dir)
	}
	return paths, remove, nil
}
