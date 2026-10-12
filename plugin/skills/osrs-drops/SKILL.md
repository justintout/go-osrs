---
name: osrs-drops
description: Answer Old School RuneScape drop and monster questions - drop rates, the chance of a drop within a kill count, how unlucky a dry streak is, expected kills to finish a set of uniques, what drops an item, and monster stats. Use when the user asks about drop rates, being dry, KC to a pet or unique, what monster drops something, or a monster's stats.
---

# OSRS drops and odds

## Tools

Use the `osrs` MCP tools when they are loaded. Otherwise run the CLI with `--json`. If `osrs` is missing, tell the user to install it with `go install github.com/justintout/go-osrs/cmd/osrs@latest`.

| Question | Tool / command |
| --- | --- |
| Odds for drops from a monster at a KC | `odds` / `osrs odds <monster> --items "Vorki,Draconic visage" --kills 4000 --json` |
| Odds for a known rate | `odds` / `osrs odds --rate 1/5000 --kills 9000 --json` |
| A monster's full drop table | `drops` / `osrs drops <monster> --json` |
| What drops an item | `drops` / `osrs drops --item "<item>" --json` |
| Monster stats | `wiki_monster` / `osrs wiki-monster <monster> --json` |
| Anything else on the wiki | `wiki_search`, then `wiki_summary` |

Monster and item names are wiki page titles. If one fails, use `wiki_search` to find the right title.

## Reading the data

- Drops are independent per kill, so a dry streak does not raise the next kill's chance. Say this when someone feels "due".
- `dry_chance` is the share of players who would still have no drop at that kill count. 0.05 means 1 in 20 players is this dry or drier.
- `expected_kills_for_all` treats the drops as independent. Many bosses roll uniques from one shared table, so the real number is close but not exact.
- Rarity text like "Rare" or "Varies" has no number; `odds` refuses those items. Quote the wiki text instead.
- A drop table can list one item several times, for different monster versions or quantities. `odds` uses the most likely line and reports its `source`.
