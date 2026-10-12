package wiki

import (
	"slices"
	"testing"
)

func TestParseQuest(t *testing.T) {
	text := `*Completion of the following quests:
**[[Enlightened Journey]]
***[[Balloon transport system#Grand Tree|Balloon flight route]] must be unlocked.
**[[Recipe for Disaster/Freeing_King_Awowogei|Recipe for Disaster]] (Freeing King [[Awowogei]])
***[[Monkey Madness I]]
*<span class="scp" data-skill="Quest points" data-level="200">200 Quest points</span>
*<span class="scp" data-skill="Slayer" data-level="69">69 [[Slayer]]</span> <sup>[<span title="This requirement is not boostable">not boostable</span>]</sup>
*<span class="scp" data-skill="Crafting" data-level="70">70 [[Crafting]]</span>`
	known := map[string]bool{"Enlightened Journey": true, "Recipe for Disaster/Freeing King Awowogei": true, "Monkey Madness I": true}
	q := parseQuest("Monkey Madness II", "Grandmaster", "Very Long", text, known)
	if !slices.Equal(q.Prerequisites, []string{"Enlightened Journey", "Recipe for Disaster/Freeing King Awowogei"}) {
		t.Errorf("prerequisites = %v", q.Prerequisites)
	}
	want := []SkillRequirement{{Skill: "Slayer", Level: 69, Boostable: false}, {Skill: "Crafting", Level: 70, Boostable: true}}
	if !slices.Equal(q.Skills, want) || q.QuestPoints != 200 {
		t.Errorf("quest = %+v", q)
	}
}
