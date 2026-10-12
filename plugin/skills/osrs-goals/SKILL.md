---
name: osrs-goals
description: Plan Old School RuneScape goals for a player - what quests they can do now, what a quest needs across its prerequisites, the cheapest or fastest way to train a skill at live GE prices, the full crafting tree an ironman needs to make an item, and XP or combat level math. Use when the user asks what to do next, how to unlock a quest, how to get a skill to a level, how much a 99 costs, what goes into making an item, or how far they are from a requirement.
---

# OSRS goal planning

## Tools

Use the `osrs` MCP tools when they are loaded. Otherwise run the CLI with `--json`. If `osrs` is missing, tell the user to install it with `go install github.com/justintout/go-osrs/cmd/osrs@latest`.

| Question | Tool / command |
| --- | --- |
| Everything a quest needs, prerequisites in order | `quest` / `osrs quest "<quest>" --player <player> --json` |
| Which quests a player has the levels for, closest blocked ones | `quests` / `osrs quests <player> --json` |
| Training plan to a level | `train` / `osrs train <skill> --player <player> --to 99 --by gp --json` |
| Everything that goes into making an item (ironman) | `craft` / `osrs craft "<item>" --quantity 100 [--format mermaid] --json` |
| XP to a level | `xp` / `osrs xp --level 70 --target 99 --json` |
| Combat level | `combat` / `osrs combat --attack 60 --strength 70 ... --json` |
| Current levels | `hiscores` / `osrs hiscores <player> --json` |

## Reading the data

- Quest completion is not public. `quests` lists quests the player has the skill levels for, which includes ones they have already done. Ask what they have finished before recommending.
- `quest` reports the highest level each skill needs anywhere in the prerequisite tree. A `boostable: false` skill must be met without boosts.
- Unranked hiscores skills read as level 1, so a blocked quest may only look blocked.
- `train --by gp` maximizes coins per XP. When methods cost money, that is the cheapest route. When some methods make money, it favors slow profitable ones, so offer `--by xp` for speed. Plans use live prices and cap each recipe at one day of market volume (`market_days`). Treat totals as estimates.
- `train` covers Woodcutting and skills trained by making items (Fletching, Smithing, Crafting, Herblore, Cooking, and similar). It cannot plan Agility, Slayer, or combat skills.

- `craft` follows wiki recipes down to raw materials and says how to get each: a gathering skill, the cheapest coin shop, or the best drop. Its `mermaid` field is a flowchart; show it in a mermaid code block when a diagram helps. When an item has no recipe, the error lists similar items that do, such as a 3-dose potion for a 4-dose one.

## Answer shape

Turn the data into a short ordered plan: what to do first, what it unlocks, and the cost or time if known. State assumptions, such as quests the player said they have done.
