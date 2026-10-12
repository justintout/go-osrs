package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/justintout/go-osrs/examples/woodcutting/views"
	"github.com/justintout/go-osrs/hiscores"
	"github.com/justintout/go-osrs/woodcutting"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		views.Index(woodcutting.Trees, nil, nil).Render(r.Context(), w)
	})

	http.HandleFunc("/calculate", func(w http.ResponseWriter, r *http.Request) {
		startLevel, err1 := strconv.Atoi(r.URL.Query().Get("startLevel"))
		targetLevel, err2 := strconv.Atoi(r.URL.Query().Get("targetLevel"))
		if err1 != nil || err2 != nil || startLevel < 1 || targetLevel > 99 || startLevel >= targetLevel {
			http.Error(w, "levels must satisfy 1 <= start < target <= 99", http.StatusBadRequest)
			return
		}
		startXp := woodcutting.XPForLevel(startLevel)
		targetXp := woodcutting.XPForLevel(targetLevel)
		disabledTrees := r.URL.Query()["disabledTrees"]
		overrideLevels := make(map[string]int)
		for _, tree := range woodcutting.Trees {
			overrideStr := r.URL.Query().Get("override_" + tree.Name)
			if overrideStr != "" {
				override, err := strconv.Atoi(overrideStr)
				if err != nil || override < tree.Level || override > 99 {
					http.Error(w, fmt.Sprintf("%s override must be between %d and 99", tree.Name, tree.Level), http.StatusBadRequest)
					return
				}
				overrideLevels[tree.Name] = override
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
