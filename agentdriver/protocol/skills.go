package driverprotocol

import (
	"fmt"
	"path"
	"regexp"
	"strings"
	"unicode"
)

const MaxSkillBytes = 4 << 20

// Skill carries trusted host-bundled content, not backend installation paths.
type Skill struct {
	Name  string      `json:"name"`
	Files []SkillFile `json:"files"`
}
type SkillFile struct {
	Path       string `json:"path"`
	Data       []byte `json:"data"`
	Executable bool   `json:"executable,omitempty"`
}

var skillName = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func ValidateSkills(skills []Skill) error {
	names := map[string]bool{}
	total := 0
	if len(skills) > 16 {
		return fmt.Errorf("too many bundled skills")
	}
	for _, skill := range skills {
		if len(skill.Name) > 64 || !skillName.MatchString(skill.Name) || names[skill.Name] {
			return fmt.Errorf("invalid or duplicate skill name %q", skill.Name)
		}
		names[skill.Name] = true
		if len(skill.Files) == 0 || len(skill.Files) > 64 {
			return fmt.Errorf("invalid skill file count")
		}
		files := map[string]bool{}
		for _, file := range skill.Files {
			p := file.Path
			if p == "." || path.IsAbs(p) || path.Clean(p) != p || strings.Contains(p, "\\") || strings.ContainsRune(p, ':') || strings.IndexFunc(p, unicode.IsControl) >= 0 || files[p] {
				return fmt.Errorf("invalid or duplicate skill path %q", p)
			}
			for _, part := range strings.Split(p, "/") {
				if part == "" || strings.HasPrefix(part, ".") {
					return fmt.Errorf("unsafe skill path %q", p)
				}
			}
			files[p] = true
			total += len(file.Data)
			if total > MaxSkillBytes {
				return fmt.Errorf("bundled skills exceed size limit")
			}
		}
		if !files["SKILL.md"] {
			return fmt.Errorf("skill %s is missing SKILL.md", skill.Name)
		}
		for p := range files {
			for parent := path.Dir(p); parent != "."; parent = path.Dir(parent) {
				if files[parent] {
					return fmt.Errorf("skill file/directory collision at %s", parent)
				}
			}
		}
	}
	return nil
}
