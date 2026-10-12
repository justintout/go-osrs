package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/justintout/go-osrs"
	"github.com/justintout/go-osrs/examples/woodcutting/views"
	"github.com/justintout/go-osrs/hiscores"
	"github.com/justintout/go-osrs/skills"
	"github.com/justintout/go-osrs/woodcutting"
)

func main() {
	hc := hiscores.NewClient("go-osrs woodcutting example (https://github.com/justintout/go-osrs)")

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
		startXp := skills.XPForLevel(startLevel)
		targetXp := skills.XPForLevel(targetLevel)
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
		player, err := hc.Lookup(r.Context(), username, hiscores.Regular)
		if errors.Is(err, osrs.ErrNotFound) {
			http.Error(w, "Player not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "Hiscores lookup failed", http.StatusBadGateway)
			return
		}
		views.PlayerInfo(player).Render(r.Context(), w)
	})

	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
