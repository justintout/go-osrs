package hiscores

import (
	"bufio"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const HiscoresURL = "https://secure.runescape.com/m=hiscore_oldschool/index_lite.ws?player=%s"

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
	resp, err := http.Get(fmt.Sprintf(HiscoresURL, username))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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