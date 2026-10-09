package pi

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	protocol "github.com/regb/workitem/agentdriver/protocol"
	"github.com/regb/workitem/skills"
)

func TestExportSkills(t *testing.T) {
	resources := skills.Bundled()
	resources[0].Files = append(resources[0].Files, protocol.SkillFile{Path: "scripts/helper.sh", Data: []byte("#!/bin/sh\n"), Executable: true})
	paths, cleanup, err := exportSkills(context.Background(), resources)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if len(paths) != 1 || !filepath.IsAbs(paths[0]) {
		t.Fatalf("paths=%v", paths)
	}
	for _, file := range resources[0].Files {
		path := filepath.Join(paths[0], filepath.FromSlash(file.Path))
		b, err := os.ReadFile(path)
		if err != nil || string(b) != string(file.Data) {
			t.Fatalf("file %s: error=%v", file.Path, err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		want := os.FileMode(0600)
		if file.Executable {
			want = 0700
		}
		if info.Mode().Perm() != want {
			t.Fatalf("%s mode=%o", file.Path, info.Mode().Perm())
		}
	}
	root := filepath.Dir(paths[0])
	info, err := os.Stat(root)
	if err != nil || info.Mode().Perm() != 0700 {
		t.Fatalf("root info=%v error=%v", info, err)
	}
	cleanup()
	cleanup()
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("temporary skill survived cleanup: %v", err)
	}
}

func TestExportSkillsValidatesBeforeWriting(t *testing.T) {
	root := t.TempDir()
	t.Setenv("TMPDIR", root)
	bad := []protocol.Skill{{Name: "wi", Files: []protocol.SkillFile{{Path: "SKILL.md"}, {Path: "../escape"}}}}
	if _, _, err := exportSkills(context.Background(), bad); err == nil {
		t.Fatal("accepted escaping path")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := exportSkills(ctx, skills.Bundled()); err != context.Canceled {
		t.Fatalf("error=%v", err)
	}
	paths, cleanup, err := exportSkills(context.Background(), nil)
	if err != nil || len(paths) != 0 {
		t.Fatalf("paths=%v error=%v", paths, err)
	}
	cleanup()
	files, err := os.ReadDir(root)
	if err != nil || len(files) != 0 {
		t.Fatalf("temporary files=%v error=%v", files, err)
	}
}

func TestExportSkillsUsesIndependentRuntimeDirectories(t *testing.T) {
	var wg sync.WaitGroup
	paths := make(chan string, 8)
	for i := 0; i < cap(paths); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, cleanup, err := exportSkills(context.Background(), skills.Bundled())
			if err != nil {
				t.Error(err)
				return
			}
			defer cleanup()
			paths <- p[0]
		}()
	}
	wg.Wait()
	close(paths)
	seen := map[string]bool{}
	for path := range paths {
		if seen[path] {
			t.Fatalf("shared runtime directory %s", path)
		}
		seen[path] = true
	}
}
