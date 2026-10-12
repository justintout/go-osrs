---
name: osrs-player
description: Look up and summarize an Old School RuneScape player from the hiscores and Wise Old Man - stats, combat level, account type (ironman, hardcore, ultimate, de-ironed), boss kill counts, recent gains, and side-by-side comparisons. Use when the user names an OSRS player or asks about someone's levels, KC, progress, rank, or account type.
---

# OSRS player lookup

## Tools

Use the `osrs` MCP tools when they are loaded. Otherwise run the CLI with `--json` and read the JSON. If `osrs` is missing, tell the user to install it with `go install github.com/justintout/go-osrs/cmd/osrs@latest`.

| Question | Tool / command |
| --- | --- |
| Levels, XP, ranks, boss KC, clues | `hiscores` / `osrs hiscores <player> --json` |
| Account type and modes left | `account` / `osrs account <player> --json` |
| Several players side by side | `compare` / `osrs compare "a, b, c" --json` |
| Recent progress | `wom_gains` / `osrs wom-gains <player> --period week --json` |
| EHP, EHB, hours to max | `wom_player` / `osrs wom-player <player> --json` |
| Top of a leaderboard | `leaderboard` / `osrs leaderboard --skill Woodcutting --json` |

`hiscores` takes a `mode`: regular, ironman, hardcore, ultimate, deadman, seasonal, tournament, fresh_start, 1_defence, or skiller.

## Reading the data

- Rank and XP of -1 mean unranked. Skills below the hiscores threshold read as level 1, so a low level may be higher in game.
- `account` compares Overall XP across boards. A player who left a mode and has not gained XP since still reads as that mode. Group ironmen are not detected.
- Hiscores update when the player logs out. Wise Old Man updates only when someone requests it; if its data looks stale or the player is untracked, run `wom_update` first, then read again. Do not call `wom_update` in a loop: Wise Old Man allows 20 requests a minute without a key.
- Activities with a score of -1 are unranked, not zero. Report only ranked activities unless asked.

## Answer shape

Lead with what the user asked. For a general summary give: account type, combat level, total level, the strongest and weakest skills, and the top boss kill counts. Keep numbers formatted with separators.
