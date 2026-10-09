package driverprotocol

import "testing"

func TestValidateSkills(t *testing.T) {
	good := Skill{Name: "wi", Files: []SkillFile{{Path: "SKILL.md", Data: []byte("skill")}, {Path: "scripts/helper.sh", Executable: true}}}
	if err := ValidateSkills([]Skill{good}); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"../outside", "/absolute", "a/../outside", ".hidden.json", "a/.hidden", "a//b", "a\\b", "a\nfile", "a\x1bfile", "a\x7ffile", "a\u0085file", "."} {
		t.Run(p, func(t *testing.T) {
			bad := good
			bad.Files = append([]SkillFile(nil), good.Files...)
			bad.Files = append(bad.Files, SkillFile{Path: p})
			if ValidateSkills([]Skill{bad}) == nil {
				t.Fatal("accepted unsafe path")
			}
		})
	}
	for _, bad := range []Skill{
		{Name: "../wi", Files: good.Files}, {Name: "wi", Files: []SkillFile{{Path: "readme"}}},
		{Name: "wi", Files: []SkillFile{{Path: "SKILL.md"}, {Path: "SKILL.md"}}},
		{Name: "wi", Files: []SkillFile{{Path: "SKILL.md"}, {Path: "scripts"}, {Path: "scripts/a"}}},
		{Name: "wi", Files: []SkillFile{{Path: "SKILL.md", Data: make([]byte, MaxSkillBytes+1)}}},
	} {
		if ValidateSkills([]Skill{bad}) == nil {
			t.Fatalf("accepted %s", bad.Name)
		}
	}
	if ValidateSkills([]Skill{good, good}) == nil {
		t.Fatal("accepted duplicate skill")
	}
}
