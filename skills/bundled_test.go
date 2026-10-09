package skills

import (
	"bytes"
	protocol "github.com/regb/workitem/agentdriver/protocol"
	"os"
	"testing"
)

func TestBundledSkillMatchesSource(t *testing.T) {
	bundles := Bundled()
	if err := protocol.ValidateSkills(bundles); err != nil {
		t.Fatal(err)
	}
	if len(bundles) != 1 || bundles[0].Name != "wi" {
		t.Fatalf("bundles=%v", bundles)
	}
	source, err := os.ReadFile("wi/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range bundles[0].Files {
		if f.Path == "SKILL.md" {
			if !bytes.Equal(f.Data, source) {
				t.Fatal("embedded skill differs from source")
			}
			return
		}
	}
	t.Fatal("no SKILL.md")
}
