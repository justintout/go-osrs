package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/justintout/osrs/hiscores"
	"github.com/justintout/osrs/woodcutting"
	"github.com/justintout/osrs/woodcutting/views"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		views.Index(woodcutting.Trees, nil, nil).Render(r.Context(), w)
	})

	http.HandleFunc("/calculate", func(w http.ResponseWriter, r *http.Request) {
		startLevel, _ := strconv.Atoi(r.URL.Query().Get("startLevel"))
		targetLevel, _ := strconv.Atoi(r.URL.Query().Get("targetLevel"))
		startXp := woodcutting.XPForLevel(startLevel)
		targetXp := woodcutting.XPForLevel(targetLevel)
		disabledTrees := r.URL.Query()["disabledTrees"]
		overrideLevels := make(map[string]int)
		for _, tree := range woodcutting.Trees {
			overrideStr := r.URL.Query().Get("override_" + tree.Name)
			if overrideStr != "" {
				if override, err := strconv.Atoi(overrideStr); err == nil {
					overrideLevels[tree.Name] = override
				}
			}
		}

		actions := woodcutting.Calculate(startXp, targetXp, disabledTrees, overrideLevels)

		views.Results(actions).Render(r.Context(), w)
	})

	http.HandleFunc("/lookup", func(w http.ResponseWriter, r *http.Request) {
		username := r.URL.Query().Get("username")
		player, err := hiscores.GetPlayer(username)
		if err != nil {
			// Handle error appropriately
			http.Error(w, "Player not found", http.StatusNotFound)
			return
		}
		views.PlayerInfo(player).Render(r.Context(), w)
	})

	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}