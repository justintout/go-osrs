package wiki

import (
	"context"
	"regexp"
	"strconv"
	"strings"
)

// Quest is a quest's direct requirements, parsed from its wiki page.
type Quest struct {
	Name       string `json:"name"`
	Difficulty string `json:"difficulty"`
	Length     string `json:"length"`
	// Skills are the skill levels the quest itself needs. Prerequisite
	// quests can need more; see the quests package.
	Skills      []SkillRequirement `json:"skills,omitempty"`
	QuestPoints int                `json:"quest_points,omitempty"`
	CombatLevel int                `json:"combat_level,omitempty"`
	// Prerequisites are the quests that must be completed first.
	Prerequisites []string `json:"prerequisites,omitempty"`
}

// SkillRequirement is a skill level a quest needs.
type SkillRequirement struct {
	Skill     string `json:"skill"`
	Level     int    `json:"level"`
	Boostable bool   `json:"boostable"`
}

// Quests returns every quest and subquest on the wiki with its parsed
// requirements.
func (c *Client) Quests(ctx context.Context) ([]Quest, error) {
	var rows []struct {
		Name         string `json:"page_name"`
		Requirements string `json:"requirements"`
		Difficulty   string `json:"official_difficulty"`
		Length       string `json:"official_length"`
	}
	q := Bucket("quest").Select("page_name", "requirements", "official_difficulty", "official_length").Limit(5000)
	if err := c.Run(ctx, q, &rows); err != nil {
		return nil, err
	}
	names := make(map[string]bool, len(rows))
	for _, r := range rows {
		names[r.Name] = true
	}
	quests := make([]Quest, len(rows))
	for i, r := range rows {
		quests[i] = parseQuest(r.Name, r.Difficulty, r.Length, r.Requirements, names)
	}
	return quests, nil
}

var (
	skillReqRE = regexp.MustCompile(`data-skill="([^"]+)" data-level="(\d+)"`)
	linkRE     = regexp.MustCompile(`\[\[([^\]|#]+)`)
)

// parseQuest reads requirement wikitext. Skill requirements are spans
// with data-skill and data-level attributes. Prerequisite quests are
// second-level bullets ("**[[Quest]]"); deeper bullets are their own
// prerequisites or notes. Only links to known quest pages count.
func parseQuest(name, difficulty, length, text string, quests map[string]bool) Quest {
	q := Quest{Name: name, Difficulty: difficulty, Length: length}
	seen := map[string]bool{}
	for _, line := range strings.Split(text, "\n") {
		for _, m := range skillReqRE.FindAllStringSubmatch(line, -1) {
			level, _ := strconv.Atoi(m[2])
			switch m[1] {
			case "Quest points", "Quest point":
				q.QuestPoints = max(q.QuestPoints, level)
			case "Combat", "Combat level":
				q.CombatLevel = max(q.CombatLevel, level)
			default:
				q.Skills = append(q.Skills, SkillRequirement{Skill: m[1], Level: level, Boostable: !strings.Contains(line, "not boostable")})
			}
		}
		if !strings.HasPrefix(line, "**") || strings.HasPrefix(line, "***") {
			continue
		}
		if m := linkRE.FindStringSubmatch(line); m != nil {
			prereq := strings.TrimSpace(strings.ReplaceAll(m[1], "_", " "))
			if quests[prereq] && prereq != name && !seen[prereq] {
				seen[prereq] = true
				q.Prerequisites = append(q.Prerequisites, prereq)
			}
		}
	}
	return q
}
