package hiscores

import (
	"bufio"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const HiscoresURL = "https://secure.runescape.com/m=hiscore_oldschool/index_lite.ws?player=%s"

// hiscoresURL is the format string actually used to build requests. It
// defaults to the live OSRS endpoint but can be overridden in tests to point
// at a local httptest server.
var hiscoresURL = HiscoresURL

type Player struct {
	Skills map[string]Skill
}

type Skill struct {
	Rank  int
	Level int
	XP    int
}

var skillNames = []string{
	"Overall", "Attack", "Defence", "Strength", "Hitpoints", "Ranged", "Prayer",
	"Magic", "Cooking", "Woodcutting", "Fletching", "Fishing", "Firemaking",
	"Crafting", "Smithing", "Mining", "Herblore", "Agility", "Thieving", "Slayer",
	"Farming", "Runecraft", "Hunter", "Construction",
}

func GetPlayer(username string) (*Player, error) {
	resp, err := http.Get(fmt.Sprintf(hiscoresURL, username))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hiscores lookup for %q failed: unexpected status code %d", username, resp.StatusCode)
	}

	player := &Player{Skills: make(map[string]Skill)}
	scanner := bufio.NewScanner(resp.Body)
	for i := 0; scanner.Scan() && i < len(skillNames); i++ {
		line := scanner.Text()
		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			continue
		}
		skillName := skillNames[i]
		rank, _ := strconv.Atoi(parts[0])
		level, _ := strconv.Atoi(parts[1])
		xp, _ := strconv.Atoi(parts[2])
		player.Skills[skillName] = Skill{Rank: rank, Level: level, XP: xp}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return player, nil
}