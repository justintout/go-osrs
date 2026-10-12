package quests

import (
	"slices"
	"testing"

	"github.com/justintout/go-osrs/wiki"
)

func TestRequirements(t *testing.T) {
	g := New([]wiki.Quest{
		{Name: "Base", Skills: []wiki.SkillRequirement{{Skill: "Agility", Level: 50, Boostable: true}}},
		{Name: "Middle", Prerequisites: []string{"Base"}, Skills: []wiki.SkillRequirement{{Skill: "Agility", Level: 50}, {Skill: "Magic", Level: 40, Boostable: true}}},
		{Name: "Top", Prerequisites: []string{"Middle", "Base"}, QuestPoints: 100, Skills: []wiki.SkillRequirement{{Skill: "Magic", Level: 75, Boostable: true}}},
	})
	r, err := g.Requirements("top")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.Prerequisites, []string{"Base", "Middle"}) {
		t.Errorf("prerequisites = %v", r.Prerequisites)
	}
	want := []wiki.SkillRequirement{{Skill: "Agility", Level: 50, Boostable: false}, {Skill: "Magic", Level: 75, Boostable: true}}
	if !slices.Equal(r.Skills, want) || r.QuestPoints != 100 {
		t.Errorf("requirements = %+v", r)
	}
	unmet := r.Unmet(map[string]int{"agility": 60, "Magic": 70})
	if len(unmet) != 1 || unmet[0].Skill != "Magic" || unmet[0].Have != 70 {
		t.Errorf("unmet = %+v", unmet)
	}
}
