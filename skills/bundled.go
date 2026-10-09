// Package skills supplies the trusted skill resources shipped with this build.
package skills

import (
	"embed"
	"io/fs"
	"sort"
	"strings"

	driverprotocol "github.com/regb/workitem/agentdriver/protocol"
)

//go:embed wi
var bundled embed.FS

func Bundled() []driverprotocol.Skill {
	skill := driverprotocol.Skill{Name: "wi"}
	// The embedded tree is fixed at compile time and contains no symlinks.
	if err := fs.WalkDir(bundled, "wi", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := bundled.ReadFile(p)
		if err != nil {
			return err
		}
		relative := strings.TrimPrefix(p, "wi/")
		skill.Files = append(skill.Files, driverprotocol.SkillFile{Path: relative, Data: data, Executable: strings.HasPrefix(relative, "scripts/")})
		return nil
	}); err != nil {
		panic(err)
	}
	sort.Slice(skill.Files, func(i, j int) bool { return skill.Files[i].Path < skill.Files[j].Path })
	return []driverprotocol.Skill{skill}
}
