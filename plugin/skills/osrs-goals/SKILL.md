---
name: osrs-goals
description: Plan Old School RuneScape goals for a player - what quests they can do now, what a quest needs across its prerequisites, the cheapest or fastest way to train a skill to a level at live GE prices, and XP or combat level math. Use when the user asks what to do next, how to unlock a quest, how to get a skill to a level, how much a 99 costs, or how far they are from a requirement.
---

# OSRS goal planning

## Tools

Use the `osrs` MCP tools when they are loaded. Otherwise run the CLI with `--json`. If `osrs` is missing, tell the user to install it with `go install github.com/justintout/go-osrs/cmd/osrs@latest`.

| Question | Tool / command |
| --- | --- |
| Everything a quest needs, prerequisites in order | `quest` / `osrs quest "<quest>" --player <player> --json` |
| Which quests a player has the levels for, closest blocked ones | `quests` / `osrs quests <player> --json` |
| Training plan to a level | `train` / `osrs train <skill> --player <player> --to 99 --by gp --json` |
| XP to a level | `xp` / `osrs xp --level 70 --target 99 --json` |
| Combat level | `combat` / `osrs combat --attack 60 --strength 70 ... --json` |
| Current levels | `hiscores` / `osrs hiscores <player> --json` |

## Reading the data

- Quest completion is not public. `quests` lists quests the player has the skill levels for, which includes ones they have already done. Ask what they have finished before recommending.
- `quest` reports the highest level each skill needs anywhere in the prerequisite tree. A `boostable: false` skill must be met without boosts.
- Unranked hiscores skills read as level 1, so a blocked quest may only look blocked.
- `train --by gp` maximizes coins per XP. When methods cost money, that is the cheapest route. When some methods make money, it favors slow profitable ones, so offer `--by xp` for speed. Plans use live prices and cap each recipe at one day of market volume (`market_days`). Treat totals as estimates.
- `train` covers Woodcutting and skills trained by making items (Fletching, Smithing, Crafting, Herblore, Cooking, and similar). It cannot plan Agility, Slayer, or combat skills.

## Answer shape

Turn the data into a short ordered plan: what to do first, what it unlocks, and the cost or time if known. State assumptions, such as quests the player said they have done.
